<?php
/**
 * Settings › ReputeCast.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

use WP_Error;

defined( 'ABSPATH' ) || exit;

/**
 * Connect (the instance's address and the site's key), the connection's
 * health, the forms, the posts sent, the key and Disconnect. Every action
 * posts to admin-post.php with a nonce, and needs manage_options.
 */
final class Settings_Page {

	const SLUG       = 'reputecast';
	const CAPABILITY = 'manage_options';

	/** Hooks in (admin only). */
	public static function register(): void {
		add_action( 'admin_menu', array( self::class, 'menu' ) );
		add_action( 'admin_enqueue_scripts', array( self::class, 'styles' ) );
		add_action( 'admin_post_reputecast_connect', array( self::class, 'connect' ) );
		add_action( 'admin_post_reputecast_disconnect', array( self::class, 'disconnect' ) );
		add_action( 'admin_post_reputecast_report', array( self::class, 'report_now' ) );
		add_filter( 'plugin_action_links_' . plugin_basename( REPUTECAST_FILE ), array( self::class, 'action_links' ) );
	}

	/** Settings › ReputeCast. */
	public static function menu(): void {
		add_options_page( __( 'ReputeCast', 'reputecast' ), __( 'ReputeCast', 'reputecast' ), self::CAPABILITY, self::SLUG, array( self::class, 'render' ) );
	}

	/** The page's address. */
	public static function url(): string {
		return admin_url( 'options-general.php?page=' . self::SLUG );
	}

	/**
	 * "Settings" on the Plugins screen.
	 *
	 * @param array<int|string, string> $links The plugin's links.
	 * @return array<int|string, string>
	 */
	public static function action_links( $links ): array {
		$settings = '<a href="' . esc_url( self::url() ) . '">' . esc_html__( 'Settings', 'reputecast' ) . '</a>';
		return array_merge( array( 'settings' => $settings ), is_array( $links ) ? $links : array() );
	}

	/**
	 * The page's few styles, on this page only.
	 *
	 * @param string $hook The admin page.
	 */
	public static function styles( $hook ): void {
		if ( 'settings_page_' . self::SLUG === $hook ) {
			wp_enqueue_style( 'reputecast-settings', plugins_url( 'assets/settings.css', REPUTECAST_FILE ), array(), REPUTECAST_VERSION );
		}
	}

	/** The language of ReputeCast's messages: the admin's. */
	private static function language(): string {
		return Languages::bcp47( get_user_locale() );
	}

	/** Connect, or save a new key (admin-post.php). */
	public static function connect(): void {
		self::guard( 'reputecast_connect' );
		// phpcs:disable WordPress.Security.NonceVerification.Missing -- checked in guard().
		$url = isset( $_POST['instance_url'] ) ? sanitize_text_field( wp_unslash( $_POST['instance_url'] ) ) : '';
		$key = isset( $_POST['site_key'] ) ? sanitize_text_field( wp_unslash( $_POST['site_key'] ) ) : '';
		// phpcs:enable
		$notice = self::do_connect( $url, $key );
		if ( 'error' === $notice['type'] ) {
			$notice['instance_url'] = $url;
		}
		self::finish( $notice );
	}

	/** Disconnect (admin-post.php). */
	public static function disconnect(): void {
		self::guard( 'reputecast_disconnect' );
		self::finish( self::do_disconnect() );
	}

	/** Report in now (admin-post.php). */
	public static function report_now(): void {
		self::guard( 'reputecast_report' );
		self::finish( self::do_report() );
	}

	/**
	 * Only administrators, and only from this page.
	 *
	 * @param string $action The nonce's action.
	 */
	private static function guard( string $action ): void {
		if ( ! current_user_can( self::CAPABILITY ) ) {
			wp_die( esc_html__( 'You can’t change the ReputeCast settings.', 'reputecast' ), '', array( 'response' => 403 ) );
		}
		check_admin_referer( $action );
	}

	/**
	 * Back to the page, with a notice.
	 *
	 * @param array<string, string> $notice { type, message }.
	 */
	private static function finish( array $notice ): void {
		set_transient( 'reputecast_notice_' . get_current_user_id(), $notice, 5 * MINUTE_IN_SECONDS );
		wp_safe_redirect( self::url() );
		exit;
	}

	/**
	 * Connects with an address and a key: checks the key, keeps it, reports
	 * in and fetches the forms.
	 *
	 * @param string $raw_url What was pasted as the address.
	 * @param string $key     What was pasted as the key.
	 * @return array{type: string, message: string}
	 */
	public static function do_connect( string $raw_url, string $key ): array {
		$url = Client::normalize_url( $raw_url );
		$key = trim( $key );
		if ( '' === $url ) {
			return self::notice( 'error', __( 'Paste your ReputeCast address, for example https://reputecast.example.com.', 'reputecast' ) );
		}
		if ( ! Client::looks_like_key( $key ) ) {
			return self::notice( 'error', __( 'That isn’t a ReputeCast key. Copy the whole key ReputeCast shows when you connect a site: it starts with rc_live_.', 'reputecast' ) );
		}
		$client = new Client( $url, $key, '', self::language() );
		$me     = $client->me();
		if ( is_wp_error( $me ) ) {
			return self::notice( 'error', $me->get_error_message() );
		}
		if ( Client::INTEGRATION !== ( $me['integration'] ?? '' ) ) {
			return self::notice( 'error', __( 'This key isn’t a WordPress site’s key. In ReputeCast, open Settings › Integrations › WordPress and choose Connect a site: it shows the site’s key.', 'reputecast' ) );
		}
		$workspace = Reporter::slug( is_string( $me['workspace'] ?? null ) ? $me['workspace'] : '' );
		if ( '' === $workspace ) {
			return self::notice( 'error', __( 'ReputeCast didn’t say which workspace the key belongs to. Check the address: use the one ReputeCast shows.', 'reputecast' ) );
		}

		$before     = Options::connection();
		$before_key = Options::key();
		$same       = $before['instance_url'] === $client->base() && $before['workspace'] === $workspace;
		$changes    = array(
			'instance_url'      => $client->base(),
			'workspace'         => $workspace,
			'workspace_name'    => is_string( $me['workspaceName'] ?? null ) ? sanitize_text_field( $me['workspaceName'] ) : '',
			'connected_at'      => $same && $before['connected_at'] ? $before['connected_at'] : time(),
			'last_report_error' => '',
		);
		if ( ! $same ) {
			// Another workspace: its forms, posts and checks start afresh.
			$changes += array(
				'site'           => '',
				'forms'          => array(),
				'forms_at'       => 0,
				'posts'          => array(),
				'last_report_at' => 0,
				'last_check_at'  => 0,
				'latest_plugin'  => '',
			);
		}
		Options::set_key( $key );
		Options::update( $changes );

		$report = Reporter::report( self::language() );
		if ( is_wp_error( $report ) && 409 === self::status_of( $report ) ) {
			// The site belongs to another workspace: put everything back.
			update_option( Options::CONNECTION, $before );
			if ( '' === $before_key ) {
				Options::forget_key();
			} else {
				Options::set_key( $before_key );
			}
			return self::notice( 'error', $report->get_error_message() );
		}
		Reporter::schedule();
		Forms::refresh( self::language() );
		$c = Options::connection();
		if ( is_wp_error( $report ) ) {
			return self::notice(
				'warning',
				/* translators: 1: the workspace's name, 2: what went wrong. */
				sprintf( __( 'Connected to %1$s, but the first report didn’t arrive: %2$s The site reports in again within 12 hours, or press Report in now.', 'reputecast' ), $c['workspace_name'], self::sentence( $report->get_error_message() ) )
			);
		}
		return self::notice(
			'success',
			/* translators: 1: the workspace's name, 2: the site's address as ReputeCast lists it. */
			sprintf( __( 'Connected to %1$s. ReputeCast lists this site as %2$s.', 'reputecast' ), $c['workspace_name'], $c['site'] )
		);
	}

	/**
	 * Forgets the key. The forms keep loading from the instance.
	 *
	 * @return array{type: string, message: string}
	 */
	public static function do_disconnect(): array {
		$c    = Options::connection();
		$name = '' !== $c['workspace_name'] ? $c['workspace_name'] : $c['workspace'];
		Options::forget_key();
		Reporter::unschedule();
		wp_unschedule_hook( Posts::HOOK );
		return self::notice(
			'success',
			/* translators: %s: the workspace's name. */
			sprintf( __( 'Disconnected from %s. Signup forms keep working. To remove the site from ReputeCast too, disconnect it there, in Settings › Integrations › WordPress.', 'reputecast' ), $name )
		);
	}

	/**
	 * Reports in, and refreshes the forms.
	 *
	 * @return array{type: string, message: string}
	 */
	public static function do_report(): array {
		$report = Reporter::report( self::language() );
		if ( is_wp_error( $report ) ) {
			return self::notice( 'error', $report->get_error_message() );
		}
		$forms = Forms::refresh( self::language() );
		$c     = Options::connection();
		if ( is_wp_error( $forms ) ) {
			return self::notice(
				'warning',
				/* translators: %s: what went wrong. */
				sprintf( __( 'Reported in, but the forms list didn’t update: %s', 'reputecast' ), $forms->get_error_message() )
			);
		}
		return self::notice(
			'success',
			/* translators: %s: the site's address as ReputeCast lists it. */
			sprintf( __( 'Reported in. ReputeCast lists this site as %s.', 'reputecast' ), $c['site'] )
		);
	}

	/**
	 * A notice for the next page load.
	 *
	 * @param string $type    success, warning or error.
	 * @param string $message The sentence.
	 * @return array{type: string, message: string}
	 */
	private static function notice( string $type, string $message ): array {
		return array(
			'type'    => $type,
			'message' => $message,
		);
	}

	/**
	 * The HTTP status of an API error.
	 *
	 * @param WP_Error $error The error.
	 */
	private static function status_of( WP_Error $error ): int {
		$data = $error->get_error_data();
		return is_array( $data ) && isset( $data['status'] ) ? (int) $data['status'] : 0;
	}

	/**
	 * A message as a sentence, ending with a full stop.
	 *
	 * @param string $s The message.
	 */
	private static function sentence( string $s ): string {
		$s = rtrim( $s );
		return '' === $s || preg_match( '/[.!?]$/u', $s ) ? $s : $s . '.';
	}

	/**
	 * "6 minutes ago", with the date and time as a tooltip.
	 *
	 * @param int $ts A time.
	 */
	private static function ago( int $ts ): string {
		$when = wp_date( get_option( 'date_format' ) . ' ' . get_option( 'time_format' ), $ts );
		/* translators: %s: how long ago, like "6 minutes". */
		$text = sprintf( __( '%s ago', 'reputecast' ), human_time_diff( $ts, time() ) );
		return '<time datetime="' . esc_attr( gmdate( 'c', $ts ) ) . '" title="' . esc_attr( (string) $when ) . '">' . esc_html( $text ) . '</time>';
	}

	/** Renders the page. */
	public static function render(): void {
		if ( ! current_user_can( self::CAPABILITY ) ) {
			return;
		}
		$notice = get_transient( 'reputecast_notice_' . get_current_user_id() );
		if ( false !== $notice ) {
			delete_transient( 'reputecast_notice_' . get_current_user_id() );
		}
		$notice = is_array( $notice ) ? $notice : array();
		echo '<div class="wrap reputecast-settings">';
		echo '<h1>' . esc_html__( 'ReputeCast', 'reputecast' ) . '</h1>';
		if ( isset( $notice['type'], $notice['message'] ) ) {
			$type = in_array( $notice['type'], array( 'success', 'warning', 'error' ), true ) ? $notice['type'] : 'info';
			echo '<div class="notice notice-' . esc_attr( $type ) . ' is-dismissible"><p>' . esc_html( (string) $notice['message'] ) . '</p></div>';
		}
		if ( Options::is_connected() ) {
			self::render_connected();
		} else {
			self::render_connect( is_string( $notice['instance_url'] ?? null ) ? $notice['instance_url'] : '' );
		}
		echo '</div>';
	}

	/**
	 * The Connect form.
	 *
	 * @param string $typed_url The address typed before an error.
	 */
	private static function render_connect( string $typed_url ): void {
		$c   = Options::connection();
		$url = '' !== $typed_url ? $typed_url : (string) $c['instance_url'];
		echo '<p class="reputecast-lead">' . esc_html__( 'Connect this site to a ReputeCast workspace to add its signup forms to your pages, and to send published posts to ReputeCast, where the site’s post-draft rule can turn them into campaign drafts.', 'reputecast' ) . '</p>';
		if ( '' !== $c['instance_url'] && '' !== $c['workspace'] ) {
			echo '<div class="notice notice-info inline"><p>';
			echo esc_html(
				sprintf(
					/* translators: 1: the workspace's name, 2: the instance's address. */
					__( 'This site was connected to %1$s at %2$s. Its signup forms keep working: they load from there.', 'reputecast' ),
					'' !== $c['workspace_name'] ? $c['workspace_name'] : $c['workspace'],
					$c['instance_url']
				)
			);
			echo '</p></div>';
		}
		echo '<form method="post" action="' . esc_url( admin_url( 'admin-post.php' ) ) . '">';
		echo '<input type="hidden" name="action" value="reputecast_connect">';
		wp_nonce_field( 'reputecast_connect' );
		echo '<table class="form-table" role="presentation"><tbody>';
		echo '<tr><th scope="row"><label for="reputecast-instance-url">' . esc_html__( 'ReputeCast address', 'reputecast' ) . '</label></th><td>';
		echo '<input type="url" id="reputecast-instance-url" name="instance_url" class="regular-text code" value="' . esc_attr( $url ) . '" placeholder="https://reputecast.example.com" required aria-describedby="reputecast-instance-url-description">';
		echo '<p class="description" id="reputecast-instance-url-description">' . esc_html__( 'The address you open ReputeCast at.', 'reputecast' ) . '</p>';
		echo '</td></tr>';
		echo '<tr><th scope="row"><label for="reputecast-site-key">' . esc_html__( 'Site key', 'reputecast' ) . '</label></th><td>';
		echo '<input type="password" id="reputecast-site-key" name="site_key" class="regular-text code" autocomplete="off" spellcheck="false" required aria-describedby="reputecast-site-key-description">';
		echo '<p class="description" id="reputecast-site-key-description">' . esc_html__( 'In ReputeCast, open Settings › Integrations › WordPress and choose Connect a site. It shows the site’s key once.', 'reputecast' ) . '</p>';
		echo '</td></tr>';
		echo '</tbody></table>';
		submit_button( __( 'Connect', 'reputecast' ) );
		echo '</form>';
		self::render_privacy();
	}

	/** The connected site: health, forms, posts, the key and Disconnect. */
	private static function render_connected(): void {
		$c    = Options::connection();
		$host = (string) wp_parse_url( $c['instance_url'], PHP_URL_HOST );

		echo '<table class="form-table" role="presentation"><tbody>';

		echo '<tr><th scope="row">' . esc_html__( 'Connection', 'reputecast' ) . '</th><td>';
		echo '<span class="reputecast-badge is-ok">' . esc_html__( 'Connected', 'reputecast' ) . '</span> ';
		printf(
			/* translators: 1: the workspace's name, 2: a link to the instance. */
			esc_html__( 'to %1$s at %2$s', 'reputecast' ),
			'<strong>' . esc_html( '' !== $c['workspace_name'] ? $c['workspace_name'] : $c['workspace'] ) . '</strong>',
			'<a href="' . esc_url( $c['instance_url'] ) . '">' . esc_html( '' !== $host ? $host : $c['instance_url'] ) . '</a>'
		);
		if ( '' !== $c['site'] ) {
			/* translators: %s: the site's address as ReputeCast lists it. */
			echo '<p class="description">' . esc_html( sprintf( __( 'ReputeCast lists this site as %s.', 'reputecast' ), $c['site'] ) ) . '</p>';
		}
		echo '</td></tr>';

		echo '<tr><th scope="row">' . esc_html__( 'Site key', 'reputecast' ) . '</th><td>';
		echo '<code>' . esc_html( Options::masked_key() ) . '</code>';
		echo '<p class="description">' . esc_html__( 'Kept encrypted. It isn’t shown again.', 'reputecast' ) . '</p>';
		echo '</td></tr>';

		echo '<tr><th scope="row">' . esc_html__( 'Last report', 'reputecast' ) . '</th><td>';
		if ( '' !== $c['last_report_error'] ) {
			echo '<span class="reputecast-badge is-error">' . esc_html__( 'Failed', 'reputecast' ) . '</span> ' . esc_html( $c['last_report_error'] );
			if ( $c['last_report_at'] ) {
				/* translators: %s: how long ago, like "6 minutes ago". */
				echo '<p class="description">' . sprintf( esc_html__( 'Last arrived %s.', 'reputecast' ), self::ago( (int) $c['last_report_at'] ) ) . '</p>'; // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped -- ago() escapes.
			}
		} elseif ( $c['last_report_at'] ) {
			echo self::ago( (int) $c['last_report_at'] ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped -- ago() escapes.
		} else {
			echo esc_html__( 'Not reported in yet.', 'reputecast' );
		}
		echo '<p class="description">' . esc_html__( 'The site reports in every 12 hours and with every post: the plugin, WordPress and PHP versions, the feed, the post categories and where forms are used.', 'reputecast' ) . '</p>';
		echo '<form method="post" action="' . esc_url( admin_url( 'admin-post.php' ) ) . '" class="reputecast-inline-form">';
		echo '<input type="hidden" name="action" value="reputecast_report">';
		wp_nonce_field( 'reputecast_report' );
		submit_button( __( 'Report in now', 'reputecast' ), 'secondary', 'submit', false );
		echo '</form>';
		echo '</td></tr>';

		echo '<tr><th scope="row">' . esc_html__( 'Plugin', 'reputecast' ) . '</th><td>';
		echo esc_html( REPUTECAST_VERSION ) . ' · ';
		if ( '' !== $c['latest_plugin'] && version_compare( $c['latest_plugin'], REPUTECAST_VERSION, '>' ) ) {
			printf(
				/* translators: 1: the newest version, 2: a link to the Plugins screen. */
				esc_html__( 'Update available: %1$s. %2$s', 'reputecast' ),
				esc_html( $c['latest_plugin'] ),
				'<a href="' . esc_url( admin_url( 'plugins.php' ) ) . '">' . esc_html__( 'Update it in Plugins.', 'reputecast' ) . '</a>'
			);
		} else {
			echo esc_html__( 'Up to date', 'reputecast' );
		}
		echo '</td></tr>';

		echo '<tr><th scope="row">' . esc_html__( 'Check now', 'reputecast' ) . '</th><td>';
		if ( $c['last_check_at'] ) {
			/* translators: %s: how long ago, like "6 minutes ago". */
			printf( esc_html__( 'ReputeCast last checked the site %s.', 'reputecast' ), self::ago( (int) $c['last_check_at'] ) ); // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped -- ago() escapes.
		} else {
			echo esc_html__( 'ReputeCast hasn’t checked the site yet.', 'reputecast' );
		}
		echo '<p class="description">';
		printf(
			/* translators: %s: the status endpoint's address. */
			esc_html__( 'Check now, in ReputeCast, calls %s. It answers only calls signed with this site’s key.', 'reputecast' ),
			'<code>' . esc_html( rest_url( Forms::REST_NAMESPACE . '/status' ) ) . '</code>'
		);
		echo '</p></td></tr>';

		echo '</tbody></table>';

		self::render_forms( $c );
		self::render_posts( $c );
		self::render_key( $c );
		self::render_disconnect();
		self::render_privacy();
	}

	/**
	 * The workspace's forms, with their shortcodes.
	 *
	 * @param array<string, mixed> $c The connection.
	 */
	private static function render_forms( array $c ): void {
		echo '<h2>' . esc_html__( 'Signup forms', 'reputecast' ) . '</h2>';
		echo '<p>' . esc_html__( 'Add a form to a page with the ReputeCast signup form block, or paste its shortcode. Forms load from ReputeCast, so changes made there show here. They keep working if you disconnect.', 'reputecast' ) . '</p>';
		$forms = Forms::cached();
		if ( array() === $forms ) {
			echo '<p><em>' . esc_html__( 'The workspace has no signup forms yet. Create one in ReputeCast › Forms, then press Report in now.', 'reputecast' ) . '</em></p>';
			return;
		}
		$site     = '' !== $c['site'] ? $c['site'] : (string) wp_parse_url( home_url(), PHP_URL_HOST );
		$statuses = array(
			'live'   => __( 'Live', 'reputecast' ),
			'paused' => __( 'Paused', 'reputecast' ),
			'draft'  => __( 'Draft', 'reputecast' ),
		);
		echo '<table class="widefat striped reputecast-table"><thead><tr>';
		echo '<th scope="col">' . esc_html__( 'Form', 'reputecast' ) . '</th>';
		echo '<th scope="col">' . esc_html__( 'Status', 'reputecast' ) . '</th>';
		echo '<th scope="col">' . esc_html__( 'Shortcode', 'reputecast' ) . '</th>';
		echo '<th scope="col">' . esc_html__( 'This site', 'reputecast' ) . '</th>';
		echo '</tr></thead><tbody>';
		foreach ( $forms as $form ) {
			echo '<tr>';
			echo '<td><strong>' . esc_html( (string) $form['name'] ) . '</strong></td>';
			echo '<td>' . esc_html( $statuses[ $form['status'] ] ?? (string) $form['status'] ) . '</td>';
			echo '<td><code>' . esc_html( '[reputecast_form id="' . $form['id'] . '"]' ) . '</code></td>';
			echo '<td>';
			if ( $form['siteAllowed'] ) {
				echo '<span class="reputecast-badge is-ok">' . esc_html__( 'Allowed', 'reputecast' ) . '</span>';
			} else {
				echo '<span class="reputecast-badge is-warning">' . esc_html__( 'Not allowed', 'reputecast' ) . '</span> ';
				/* translators: %s: the site's address. */
				echo esc_html( sprintf( __( '%s isn’t one of the form’s allowed domains, so its signups are rejected. Add it in ReputeCast › Forms.', 'reputecast' ), $site ) );
			}
			echo '</td></tr>';
		}
		echo '</tbody></table>';
		if ( $c['forms_at'] ) {
			/* translators: %s: how long ago, like "6 minutes ago". */
			echo '<p class="description">' . sprintf( esc_html__( 'Updated %s.', 'reputecast' ), self::ago( (int) $c['forms_at'] ) ) . '</p>'; // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped -- ago() escapes.
		}
	}

	/**
	 * The posts sent, and what ReputeCast did with them.
	 *
	 * @param array<string, mixed> $c The connection.
	 */
	private static function render_posts( array $c ): void {
		echo '<h2>' . esc_html__( 'Post drafts', 'reputecast' ) . '</h2>';
		echo '<p>' . esc_html__( 'Each post you publish goes to ReputeCast. The site’s post-draft rule, set there, decides which become a campaign draft, with its audience, template and sender. Drafts only: nobody gets anything until someone reviews and sends it in ReputeCast.', 'reputecast' );
		if ( '' !== Languages::plugin() ) {
			echo ' ' . esc_html__( 'A post’s translations go with it and become the draft’s language versions.', 'reputecast' );
		}
		echo '</p>';
		$posts = is_array( $c['posts'] ) ? $c['posts'] : array();
		if ( array() === $posts ) {
			echo '<p><em>' . esc_html__( 'No posts sent yet.', 'reputecast' ) . '</em></p>';
			return;
		}
		echo '<table class="widefat striped reputecast-table"><thead><tr>';
		echo '<th scope="col">' . esc_html__( 'Post', 'reputecast' ) . '</th>';
		echo '<th scope="col">' . esc_html__( 'Sent', 'reputecast' ) . '</th>';
		echo '<th scope="col">' . esc_html__( 'In ReputeCast', 'reputecast' ) . '</th>';
		echo '</tr></thead><tbody>';
		foreach ( $posts as $entry ) {
			$link = get_edit_post_link( (int) $entry['post'] );
			echo '<tr><td>';
			echo $link ? '<a href="' . esc_url( $link ) . '">' . esc_html( (string) $entry['title'] ) . '</a>' : esc_html( (string) $entry['title'] );
			echo '</td><td>' . self::ago( (int) $entry['at'] ) . '</td><td>'; // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped -- ago() escapes.
			echo wp_kses( self::outcome( $entry, $c ), array( 'a' => array( 'href' => array() ) ) );
			echo '</td></tr>';
		}
		echo '</tbody></table>';
	}

	/**
	 * What ReputeCast did with a post, as a sentence (HTML: a link to the draft).
	 *
	 * @param array<string, mixed> $entry What Posts::remember() kept.
	 * @param array<string, mixed> $c     The connection.
	 */
	public static function outcome( array $entry, array $c ): string {
		$draft = '';
		if ( '' !== ( $entry['campaignId'] ?? '' ) ) {
			$href  = $c['instance_url'] . '/w/' . rawurlencode( (string) $c['workspace'] ) . '/campaigns/' . rawurlencode( (string) $entry['campaignId'] ) . '/edit';
			$draft = ' <a href="' . esc_url( $href ) . '">' . esc_html__( 'Open the draft', 'reputecast' ) . '</a>';
		}
		if ( '' !== ( $entry['error'] ?? '' ) ) {
			/* translators: %s: what went wrong. */
			return esc_html( sprintf( __( 'Not sent: %s', 'reputecast' ), $entry['error'] ) );
		}
		if ( ! empty( $entry['created'] ) ) {
			return esc_html__( 'Draft created.', 'reputecast' ) . $draft;
		}
		$reasons = array(
			'exists'   => __( 'It already has a draft.', 'reputecast' ),
			'no_rule'  => __( 'No draft: the site has no post-draft rule yet. Set one in ReputeCast › Settings › Integrations › WordPress.', 'reputecast' ),
			'rule_off' => __( 'No draft: the site’s post-draft rule is off.', 'reputecast' ),
			'category' => __( 'No draft: the post isn’t in one of the rule’s categories.', 'reputecast' ),
			'no_title' => __( 'Not sent: the post has no title.', 'reputecast' ),
		);
		return esc_html( $reasons[ $entry['reason'] ?? '' ] ?? __( 'No draft.', 'reputecast' ) ) . $draft;
	}

	/**
	 * Change the key, after a rotation in ReputeCast.
	 *
	 * @param array<string, mixed> $c The connection.
	 */
	private static function render_key( array $c ): void {
		echo '<h2>' . esc_html__( 'Change the key', 'reputecast' ) . '</h2>';
		echo '<p>' . esc_html__( 'After you rotate the site’s key in ReputeCast, paste the new one here. The old key keeps working for 24 hours.', 'reputecast' ) . '</p>';
		echo '<form method="post" action="' . esc_url( admin_url( 'admin-post.php' ) ) . '">';
		echo '<input type="hidden" name="action" value="reputecast_connect">';
		echo '<input type="hidden" name="instance_url" value="' . esc_attr( (string) $c['instance_url'] ) . '">';
		wp_nonce_field( 'reputecast_connect' );
		echo '<table class="form-table" role="presentation"><tbody><tr>';
		echo '<th scope="row"><label for="reputecast-new-key">' . esc_html__( 'New site key', 'reputecast' ) . '</label></th>';
		echo '<td><input type="password" id="reputecast-new-key" name="site_key" class="regular-text code" autocomplete="off" spellcheck="false" required></td>';
		echo '</tr></tbody></table>';
		submit_button( __( 'Save the new key', 'reputecast' ), 'secondary' );
		echo '</form>';
	}

	/** Disconnect. */
	private static function render_disconnect(): void {
		echo '<h2>' . esc_html__( 'Disconnect', 'reputecast' ) . '</h2>';
		echo '<p>' . esc_html__( 'Signup forms keep working. Posts stop going to ReputeCast, and the site stops reporting in. ReputeCast keeps the site’s key until you disconnect the site there too.', 'reputecast' ) . '</p>';
		echo '<form method="post" action="' . esc_url( admin_url( 'admin-post.php' ) ) . '">';
		echo '<input type="hidden" name="action" value="reputecast_disconnect">';
		wp_nonce_field( 'reputecast_disconnect' );
		submit_button( __( 'Disconnect', 'reputecast' ), 'delete' );
		echo '</form>';
	}

	/** What leaves the site, and where to. */
	private static function render_privacy(): void {
		echo '<h2>' . esc_html__( 'What the plugin sends', 'reputecast' ) . '</h2>';
		echo '<p>' . esc_html__( 'The plugin talks only to the ReputeCast address above. It sends the site’s address and name, the plugin, WordPress and PHP versions, the feed URL, the post categories and where signup forms are used; and, for each post you publish, its title, excerpt, link, featured image, categories and language. It sends nothing about visitors or users. Signup forms load from ReputeCast in the visitor’s browser, as its embed code does.', 'reputecast' ) . '</p>';
	}
}
