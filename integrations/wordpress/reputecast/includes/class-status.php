<?php
/**
 * The status endpoint ReputeCast's "Check now" calls.
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
 * GET /wp-json/reputecast/v1/status answers the workspace the site is
 * connected to, and nobody else: ReputeCast signs the call with the site's
 * key. The signature is the hex HMAC-SHA256 of "{timestamp}.status", keyed
 * with the SHA-256 digest of the key. During a key rotation the call
 * carries one signature per live key, comma-separated.
 */
final class Status {

	/** A call more than this many seconds off is refused. */
	const WINDOW = 300;

	/** Registers the route (on rest_api_init). */
	public static function register_routes(): void {
		register_rest_route(
			Forms::REST_NAMESPACE,
			'/status',
			array(
				'methods'             => WP_REST_Server::READABLE,
				'callback'            => array( self::class, 'answer' ),
				'permission_callback' => array( self::class, 'permission' ),
			)
		);
	}

	/**
	 * Only a call signed with the site's key gets an answer.
	 *
	 * @param WP_REST_Request $request The call.
	 * @return true|WP_Error
	 */
	public static function permission( WP_REST_Request $request ) {
		$ts  = (string) $request->get_header( 'X-ReputeCast-Timestamp' );
		$sig = (string) $request->get_header( 'X-ReputeCast-Signature' );
		if ( self::verify( $ts, $sig, Options::key(), time() ) ) {
			return true;
		}
		return new WP_Error(
			'reputecast_unsigned',
			__( 'Only the ReputeCast workspace this site is connected to can check it.', 'reputecast' ),
			array( 'status' => 401 )
		);
	}

	/** The answer: the plugin, WordPress and PHP versions. */
	public static function answer(): WP_REST_Response {
		Options::update( array( 'last_check_at' => time() ) );
		$response = new WP_REST_Response(
			array(
				'plugin'    => REPUTECAST_VERSION,
				'wordpress' => (string) get_bloginfo( 'version' ),
				'php'       => PHP_MAJOR_VERSION . '.' . PHP_MINOR_VERSION,
			)
		);
		$response->header( 'Cache-Control', 'no-store' );
		return $response;
	}

	/**
	 * The signature of a call, as ReputeCast computes it.
	 *
	 * @param string $key The site's key.
	 * @param string $ts  The call's timestamp (Unix seconds).
	 */
	public static function signature( string $key, string $ts ): string {
		return hash_hmac( 'sha256', $ts . '.status', hash( 'sha256', $key, true ) );
	}

	/**
	 * Whether a call is signed with the key, within five minutes of now.
	 *
	 * @param string $ts     X-ReputeCast-Timestamp.
	 * @param string $header X-ReputeCast-Signature: one or more, comma-separated.
	 * @param string $key    The site's key ('' when disconnected).
	 * @param int    $now    The time now.
	 */
	public static function verify( string $ts, string $header, string $key, int $now ): bool {
		$ts = trim( $ts );
		if ( '' === $key || 1 !== preg_match( '/^\d{1,12}$/', $ts ) || abs( $now - (int) $ts ) > self::WINDOW ) {
			return false;
		}
		$want = self::signature( $key, $ts );
		foreach ( array_slice( explode( ',', $header ), 0, 10 ) as $sig ) {
			if ( hash_equals( $want, strtolower( trim( $sig ) ) ) ) {
				return true;
			}
		}
		return false;
	}
}
