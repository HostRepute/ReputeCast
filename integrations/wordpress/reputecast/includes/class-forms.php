<?php
/**
 * Signup forms: the block, the shortcode and the editor's list.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

use WP_Error;
use WP_REST_Request;
use WP_REST_Response;
use WP_REST_Server;

defined( 'ABSPATH' ) || exit;

/**
 * A form loads through its public script, as ReputeCast's embed code does:
 * the page gets an empty container and the script fills it. The plugin
 * keeps the workspace's forms list, so forms keep working after the site is
 * disconnected.
 */
final class Forms {

	const BLOCK          = 'reputecast/signup-form';
	const SHORTCODE      = 'reputecast_form';
	const REST_NAMESPACE = 'reputecast/v1';

	/** The editor's list refreshes from ReputeCast when it is older than this (seconds). */
	const STALE_AFTER = 300;

	/** Registers the block and the shortcode (on init). */
	public static function register(): void {
		register_block_type( REPUTECAST_DIR . '/blocks/signup-form' );
		add_shortcode( self::SHORTCODE, array( self::class, 'shortcode' ) );
	}

	/** The editor's list of forms (on rest_api_init). */
	public static function register_routes(): void {
		register_rest_route(
			self::REST_NAMESPACE,
			'/forms',
			array(
				'methods'             => WP_REST_Server::READABLE,
				'callback'            => array( self::class, 'rest_forms' ),
				'permission_callback' => static function () {
					return current_user_can( 'edit_posts' );
				},
				'args'                => array(
					'refresh' => array(
						'type'    => 'boolean',
						'default' => false,
					),
				),
			)
		);
	}

	/**
	 * A form's id as ReputeCast writes it ("frm_8c2e1a"), or '' when it isn't one.
	 *
	 * @param string $id What a block, a shortcode or ReputeCast gave.
	 */
	public static function clean_form_id( string $id ): string {
		$id = trim( $id );
		return 1 === preg_match( '/^frm_[A-Za-z0-9]{1,36}$/', $id ) ? $id : '';
	}

	/**
	 * The forms list the plugin keeps.
	 *
	 * @return array<int, array<string, mixed>>
	 */
	public static function cached(): array {
		$forms = Options::connection()['forms'];
		return is_array( $forms ) ? $forms : array();
	}

	/**
	 * A form of the list, or null.
	 *
	 * @param string $id The form's id.
	 * @return array<string, mixed>|null
	 */
	public static function find( string $id ): ?array {
		foreach ( self::cached() as $form ) {
			if ( is_array( $form ) && ( $form['id'] ?? '' ) === $id ) {
				return $form;
			}
		}
		return null;
	}

	/**
	 * Fetches the workspace's forms from ReputeCast and keeps them.
	 *
	 * @param string $language The language of ReputeCast's messages.
	 * @return array<int, array<string, mixed>>|WP_Error
	 */
	public static function refresh( string $language = '' ) {
		$client = Client::connected( $language );
		if ( null === $client ) {
			return new WP_Error( 'reputecast_not_connected', __( 'This site isn’t connected to ReputeCast.', 'reputecast' ) );
		}
		$list = $client->forms();
		if ( is_wp_error( $list ) ) {
			return $list;
		}
		$forms = array();
		foreach ( is_array( $list['items'] ?? null ) ? $list['items'] : array() as $item ) {
			$form = self::sanitize_form( $item );
			if ( null !== $form ) {
				$forms[] = $form;
			}
		}
		Options::update(
			array(
				'forms'    => $forms,
				'forms_at' => time(),
			)
		);
		return $forms;
	}

	/**
	 * A form of ReputeCast's list, with only what the plugin uses.
	 *
	 * @param mixed $item One of the list's items.
	 * @return array<string, mixed>|null
	 */
	public static function sanitize_form( $item ): ?array {
		if ( ! is_array( $item ) ) {
			return null;
		}
		$id = self::clean_form_id( is_string( $item['id'] ?? null ) ? $item['id'] : '' );
		if ( '' === $id ) {
			return null;
		}
		$status = is_string( $item['status'] ?? null ) && in_array( $item['status'], array( 'draft', 'live', 'paused' ), true ) ? $item['status'] : 'live';
		$name   = is_string( $item['name'] ?? null ) ? sanitize_text_field( $item['name'] ) : '';
		$script = is_string( $item['script'] ?? null ) ? esc_url_raw( $item['script'], array( 'http', 'https' ) ) : '';
		return array(
			'id'          => $id,
			'name'        => '' === $name ? $id : $name,
			'status'      => $status,
			'kind'        => is_string( $item['kind'] ?? null ) ? sanitize_key( $item['kind'] ) : 'embedded',
			'siteAllowed' => true === ( $item['siteAllowed'] ?? false ),
			'script'      => $script,
		);
	}

	/**
	 * A form's public script: from the list, else from the instance's
	 * address (a form added to a page after the list was fetched).
	 *
	 * @param string $id The form's id.
	 */
	public static function script_url( string $id ): string {
		$form = self::find( $id );
		if ( null !== $form && is_string( $form['script'] ?? null ) && '' !== $form['script'] ) {
			return $form['script'];
		}
		$c = Options::connection();
		if ( '' === $c['instance_url'] || '' === $c['workspace'] ) {
			return '';
		}
		return $c['instance_url'] . '/p/' . rawurlencode( $c['workspace'] ) . '/forms/' . rawurlencode( $id ) . '.js';
	}

	/**
	 * A form's language on the page: the block's setting, else the page's
	 * language from WPML or Polylang. '' leaves it to the page's lang.
	 *
	 * @param string $setting The block's language setting ('' to match the page).
	 */
	public static function language( string $setting ): string {
		$lang = Languages::bcp47( $setting );
		return '' !== $lang ? $lang : Languages::current();
	}

	/**
	 * Loads a form's script and returns its container's attributes. An empty
	 * array for a popup form, which opens by itself; null when there is no
	 * form to show.
	 *
	 * @param string $form_id  The form.
	 * @param string $language The block's language setting.
	 * @return array<string, string>|null
	 */
	public static function prepare( string $form_id, string $language = '' ): ?array {
		$id = self::clean_form_id( $form_id );
		if ( '' === $id ) {
			return null;
		}
		$script = self::script_url( $id );
		if ( '' === $script ) {
			return null;
		}
		$load = array(
			'strategy'  => 'async',
			'in_footer' => true,
		);
		// The form's script is versioned by ReputeCast; a "ver" argument would only split its cache.
		wp_enqueue_script( 'reputecast-form-' . $id, $script, array(), null, $load ); // phpcs:ignore WordPress.WP.EnqueuedResourceParameters.MissingVersion
		$form = self::find( $id );
		if ( null !== $form && 'popup' === ( $form['kind'] ?? '' ) ) {
			return array();
		}
		$attrs = array( 'data-reputecast-form' => $id );
		$lang  = self::language( $language );
		if ( '' !== $lang ) {
			$attrs['lang'] = $lang;
		}
		return $attrs;
	}

	/**
	 * [reputecast_form id="frm_8c2e1a"].
	 *
	 * @param array<string, string>|string $atts The shortcode's attributes.
	 */
	public static function shortcode( $atts ): string {
		$atts  = shortcode_atts( array( 'id' => '' ), is_array( $atts ) ? $atts : array(), self::SHORTCODE );
		$attrs = self::prepare( (string) $atts['id'] );
		if ( empty( $attrs ) ) {
			return '';
		}
		$html = '<div class="reputecast-form"';
		foreach ( $attrs as $name => $value ) {
			$html .= ' ' . $name . '="' . esc_attr( $value ) . '"';
		}
		return $html . '></div>';
	}

	/**
	 * GET /wp-json/reputecast/v1/forms: the block's list, refreshed from
	 * ReputeCast when it is older than five minutes or asked to.
	 *
	 * @param WP_REST_Request $request The request.
	 */
	public static function rest_forms( WP_REST_Request $request ): WP_REST_Response {
		$c     = Options::connection();
		$error = '';
		if ( Options::is_connected() && ( (bool) $request->get_param( 'refresh' ) || time() - (int) $c['forms_at'] > self::STALE_AFTER ) ) {
			$fresh = self::refresh( Languages::bcp47( get_user_locale() ) );
			if ( is_wp_error( $fresh ) ) {
				$error = $fresh->get_error_message();
			}
			$c = Options::connection();
		}
		$forms = array();
		foreach ( self::cached() as $form ) {
			$forms[] = array(
				'id'          => (string) $form['id'],
				'name'        => (string) $form['name'],
				'status'      => (string) $form['status'],
				'kind'        => (string) $form['kind'],
				'siteAllowed' => (bool) $form['siteAllowed'],
			);
		}
		$host = '' !== $c['site'] ? $c['site'] : (string) wp_parse_url( home_url(), PHP_URL_HOST );
		return new WP_REST_Response(
			array(
				'connected'      => Options::is_connected(),
				'site'           => $host,
				'workspaceName'  => (string) $c['workspace_name'],
				'forms'          => $forms,
				'error'          => $error,
				'canManage'      => current_user_can( 'manage_options' ),
				'settingsUrl'    => admin_url( 'options-general.php?page=' . Settings_Page::SLUG ),
				'languages'      => Languages::site_languages(),
				'languagePlugin' => Languages::plugin(),
			)
		);
	}
}
