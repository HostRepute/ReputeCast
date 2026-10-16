<?php
/**
 * The calls the plugin makes to ReputeCast.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

use WP_Error;

defined( 'ABSPATH' ) || exit;

/**
 * ReputeCast's public API, with the site's key. The plugin calls only the
 * instance it is connected to: nothing about the site goes anywhere else.
 * Errors come back as WP_Error with ReputeCast's own sentence (RFC 9457).
 */
final class Client {

	/** How ReputeCast's call log names the plugin: "WordPress, ReputeCast plugin 1.3.0". */
	const USER_AGENT_PREFIX = 'ReputeCast-WordPress/';

	/** The integration a site's key belongs to, as GET /integration names it. */
	const INTEGRATION = 'wordpress'; // phpcs:ignore WordPress.WP.CapitalPDangit.MisspelledInText -- an API value, not the name.

	/**
	 * Base URL of the instance, without /api/v1.
	 *
	 * @var string
	 */
	private string $base;

	/**
	 * The site's key.
	 *
	 * @var string
	 */
	private string $key;

	/**
	 * The key's workspace (its slug).
	 *
	 * @var string
	 */
	private string $workspace;

	/**
	 * Language of the messages ReputeCast answers with ("vi", "en-US").
	 *
	 * @var string
	 */
	private string $language;

	/**
	 * A client for one instance and key.
	 *
	 * @param string $instance_url The instance's address.
	 * @param string $key          The site's key.
	 * @param string $workspace    The key's workspace, when known.
	 * @param string $language     The language of ReputeCast's messages.
	 */
	public function __construct( string $instance_url, string $key, string $workspace = '', string $language = '' ) {
		$this->base      = self::normalize_url( $instance_url );
		$this->key       = $key;
		$this->workspace = $workspace;
		$this->language  = $language;
	}

	/**
	 * The client of the connected site, or null.
	 *
	 * @param string $language The language of ReputeCast's messages.
	 */
	public static function connected( string $language = '' ): ?self {
		if ( ! Options::is_connected() ) {
			return null;
		}
		$c = Options::connection();
		return new self( $c['instance_url'], Options::key(), $c['workspace'], $language );
	}

	/**
	 * The instance's address as typed: a scheme is added when missing, and a
	 * pasted API path or trailing slash is dropped. '' when it isn't a web
	 * address.
	 *
	 * @param string $raw What was pasted.
	 */
	public static function normalize_url( string $raw ): string {
		$url = trim( $raw );
		if ( '' === $url ) {
			return '';
		}
		if ( ! preg_match( '#^https?://#i', $url ) ) {
			$url = 'https://' . $url;
		}
		$url   = rtrim( preg_replace( '#/api/v1/?$#i', '', rtrim( $url, '/' ) ), '/' );
		$parts = wp_parse_url( $url );
		if ( ! is_array( $parts ) || empty( $parts['host'] ) || ! in_array( strtolower( $parts['scheme'] ?? '' ), array( 'http', 'https' ), true ) ) {
			return '';
		}
		return esc_url_raw( $url, array( 'http', 'https' ) );
	}

	/**
	 * Whether a key looks like one ReputeCast makes ("rc_live_…").
	 *
	 * @param string $key What was pasted.
	 */
	public static function looks_like_key( string $key ): bool {
		return 1 === preg_match( '/^rc_[a-z]+_[A-Za-z0-9]{16,200}$/', $key );
	}

	/** The instance's address. */
	public function base(): string {
		return $this->base;
	}

	/**
	 * Test connection: the key's workspace, integration and what it may do.
	 *
	 * @return array<string, mixed>|WP_Error
	 */
	public function me() {
		return $this->request( 'GET', '/integration' );
	}

	/**
	 * Report in: versions, the feed, categories and where forms are used.
	 *
	 * @param array<string, mixed> $report The report (Reporter::payload()).
	 * @return array<string, mixed>|WP_Error
	 */
	public function report( array $report ) {
		return $this->request( 'POST', $this->ws( '/wordpress/report' ), $report );
	}

	/**
	 * The workspace's signup forms, with whether this site may submit them.
	 *
	 * @return array<string, mixed>|WP_Error
	 */
	public function forms() {
		return $this->request( 'GET', $this->ws( '/wordpress/forms' ) );
	}

	/**
	 * A published post, for ReputeCast's post-draft rule to decide on.
	 *
	 * @param array<string, mixed> $post The post (Posts::payload()).
	 * @return array<string, mixed>|WP_Error
	 */
	public function draft( array $post ) {
		return $this->request( 'POST', $this->ws( '/wordpress/drafts' ), array( 'post' => $post ) );
	}

	/**
	 * A workspace path.
	 *
	 * @param string $path The path under the workspace.
	 */
	private function ws( string $path ): string {
		return '/workspaces/' . rawurlencode( $this->workspace ) . $path;
	}

	/**
	 * Calls the API.
	 *
	 * @param string                    $method HTTP method.
	 * @param string                    $path   Path under /api/v1.
	 * @param array<string, mixed>|null $body   JSON body.
	 * @return array<string, mixed>|WP_Error
	 */
	private function request( string $method, string $path, ?array $body = null ) {
		if ( '' === $this->base ) {
			return new WP_Error( 'reputecast_no_instance', __( 'Paste your ReputeCast address first, for example https://reputecast.acme.io.', 'reputecast' ) );
		}
		$headers = array(
			'Authorization' => 'Bearer ' . $this->key,
			'Accept'        => 'application/json',
		);
		if ( '' !== $this->language ) {
			$headers['Accept-Language'] = $this->language;
		}
		$args = array(
			'method'      => $method,
			'timeout'     => 15,
			// The key goes only where it was pasted, never to a redirect.
			'redirection' => 0,
			'user-agent'  => self::USER_AGENT_PREFIX . REPUTECAST_VERSION,
			'headers'     => $headers,
		);
		if ( null !== $body ) {
			$args['headers']['Content-Type'] = 'application/json';
			$args['body']                    = wp_json_encode( $body );
		}
		$response = wp_remote_request( $this->base . '/api/v1' . $path, $args );
		if ( is_wp_error( $response ) ) {
			return new WP_Error(
				'reputecast_unreachable',
				/* translators: 1: the instance's address, 2: what went wrong. */
				sprintf( __( 'ReputeCast didn’t answer at %1$s: %2$s', 'reputecast' ), $this->base, $response->get_error_message() )
			);
		}
		$status = (int) wp_remote_retrieve_response_code( $response );
		$data   = json_decode( (string) wp_remote_retrieve_body( $response ), true );
		if ( $status >= 400 ) {
			$problem = is_array( $data ) ? $data : array();
			$code    = isset( $problem['code'] ) && is_string( $problem['code'] ) ? $problem['code'] : 'http_' . $status;
			return new WP_Error(
				'reputecast_' . $code,
				self::message( $status, $problem ),
				array(
					'status'  => $status,
					'problem' => $problem,
				)
			);
		}
		if ( $status >= 300 ) {
			/* translators: %d: an HTTP status code. */
			return new WP_Error( 'reputecast_redirect', sprintf( __( 'ReputeCast answered with a redirect (HTTP %d). Check the address: use the one ReputeCast shows, with https.', 'reputecast' ), $status ) );
		}
		return is_array( $data ) ? $data : array();
	}

	/**
	 * A problem's sentence: its detail, else its title.
	 *
	 * @param int                  $status  HTTP status.
	 * @param array<string, mixed> $problem RFC 9457 problem.
	 */
	public static function message( int $status, array $problem ): string {
		foreach ( array( 'detail', 'title' ) as $field ) {
			if ( isset( $problem[ $field ] ) && is_string( $problem[ $field ] ) && '' !== trim( $problem[ $field ] ) ) {
				return trim( $problem[ $field ] );
			}
		}
		/* translators: %d: an HTTP status code. */
		return sprintf( __( 'ReputeCast answered HTTP %d.', 'reputecast' ), $status );
	}
}
