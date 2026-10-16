<?php
/**
 * What the plugin keeps: the connection, and the site's key.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

defined( 'ABSPATH' ) || exit;

/**
 * The connection lives in one option; the site's key in another, encrypted
 * with a key derived from the site's salts (wp-config.php), so a copy of
 * the database alone doesn't reveal it. The key is never shown again after
 * it is pasted, only its first and last characters.
 */
final class Options {

	const CONNECTION = 'reputecast_connection';
	const KEY        = 'reputecast_key';

	/**
	 * The connection: instance URL, workspace, site address as ReputeCast
	 * lists it, the last report, the forms list.
	 *
	 * @return array<string, mixed>
	 */
	public static function connection(): array {
		$defaults = array(
			'instance_url'          => '',
			'workspace'             => '',
			'workspace_name'        => '',
			'site'                  => '',
			'connected_at'          => 0,
			'last_report_at'        => 0,
			'last_report_error'     => '',
			'last_report_failed_at' => 0,
			'latest_plugin'         => '',
			'forms'                 => array(),
			'forms_at'              => 0,
			'posts'                 => array(),
			'last_check_at'         => 0,
		);
		$stored   = get_option( self::CONNECTION, array() );
		return array_merge( $defaults, is_array( $stored ) ? $stored : array() );
	}

	/**
	 * Changes some of the connection's values.
	 *
	 * @param array<string, mixed> $changes The values to change.
	 */
	public static function update( array $changes ): void {
		update_option( self::CONNECTION, array_merge( self::connection(), $changes ) );
	}

	/** Whether the site has a key and knows its workspace. */
	public static function is_connected(): bool {
		$c = self::connection();
		return '' !== self::key() && '' !== $c['instance_url'] && '' !== $c['workspace'];
	}

	/** The site's key, or '' when there is none or it can't be read. */
	public static function key(): string {
		$stored = get_option( self::KEY, '' );
		if ( ! is_string( $stored ) || ! str_starts_with( $stored, 'v1:' ) ) {
			return '';
		}
		$raw = base64_decode( substr( $stored, 3 ), true ); // phpcs:ignore WordPress.PHP.DiscouragedPHPFunctions.obfuscation_base64_decode -- the stored ciphertext.
		if ( false === $raw || strlen( $raw ) <= SODIUM_CRYPTO_SECRETBOX_NONCEBYTES ) {
			return '';
		}
		$nonce = substr( $raw, 0, SODIUM_CRYPTO_SECRETBOX_NONCEBYTES );
		$box   = substr( $raw, SODIUM_CRYPTO_SECRETBOX_NONCEBYTES );
		try {
			$key = sodium_crypto_secretbox_open( $box, $nonce, self::secret() );
		} catch ( \SodiumException $e ) {
			return '';
		}
		return false === $key ? '' : $key;
	}

	/**
	 * Keeps the site's key, encrypted. It isn't autoloaded.
	 *
	 * @param string $key The key ReputeCast showed once.
	 */
	public static function set_key( string $key ): void {
		$nonce = random_bytes( SODIUM_CRYPTO_SECRETBOX_NONCEBYTES );
		$box   = sodium_crypto_secretbox( $key, $nonce, self::secret() );
		update_option( self::KEY, 'v1:' . base64_encode( $nonce . $box ), false ); // phpcs:ignore WordPress.PHP.DiscouragedPHPFunctions.obfuscation_base64_encode -- the stored ciphertext.
	}

	/** Forgets the site's key. The connection's other values stay, so forms keep loading. */
	public static function forget_key(): void {
		delete_option( self::KEY );
	}

	/** The key as lists show it: "rc_live_5d1e…7b20". */
	public static function masked_key(): string {
		$key = self::key();
		if ( strlen( $key ) < 20 ) {
			return '' === $key ? '' : '••••';
		}
		$prefix = strpos( $key, '_', 3 );
		$head   = false === $prefix ? 8 : $prefix + 1;
		return substr( $key, 0, $head + 4 ) . '…' . substr( $key, -4 );
	}

	/** The encryption key, derived from the site's salts. */
	private static function secret(): string {
		return hash_hmac( 'sha256', 'reputecast-site-key', wp_salt( 'auth' ), true );
	}

	/** Removes everything the plugin keeps (uninstall). */
	public static function delete_all(): void {
		delete_option( self::CONNECTION );
		delete_option( self::KEY );
	}
}
