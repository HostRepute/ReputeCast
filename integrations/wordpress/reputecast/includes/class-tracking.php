<?php
/**
 * Site tracking: ReputeCast's tracker on the site's pages, the products
 * shown and added to the cart, and the signed-in customer.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

defined( 'ABSPATH' ) || exit;

/**
 * With a tracked site's id saved (Settings › ReputeCast › Site tracking,
 * from ReputeCast's Settings › Integrations › Website tracking), every page
 * loads the tracker, /track/v1/rc.js, from the connected ReputeCast. Then:
 *
 * - a WooCommerce product page calls reputecast('product', {id});
 * - a product added to the cart (woocommerce_add_to_cart, often in an AJAX
 *   call that renders no page) is kept in the shopper's session and called
 *   reputecast('cart', {id, quantity}) on the next page;
 * - a signed-in customer (not the site's staff) is identified by a
 *   signature of their address made with the site's identify secret, on
 *   the server: reputecast('identify', {email, signature}).
 *
 * The tracker waits for the visitor's consent (the site's consent banner
 * calls reputecast('consent', true)) unless the site is set not to, as it is
 * in ReputeCast; a browser sending Global Privacy Control is never tracked.
 */
final class Tracking {

	/** The script's handle. */
	const HANDLE = 'reputecast-tracker';

	/** The add-to-carts waiting for the next page, in the WooCommerce session. */
	const SESSION   = 'reputecast_tracker_carts';
	const MAX_CARTS = 20;

	/** Lets the page call reputecast() before the script has loaded. */
	const STUB = 'window.reputecast=window.reputecast||function(){(reputecast.q=reputecast.q||[]).push(arguments)};';

	/** Hooks in. */
	public static function register(): void {
		add_action( 'wp_enqueue_scripts', array( self::class, 'enqueue' ) );
		add_filter( 'script_loader_tag', array( self::class, 'tag' ), 10, 3 );
		add_action( 'wp_footer', array( self::class, 'footer' ), 20 );
		add_action( 'woocommerce_add_to_cart', array( self::class, 'added' ), 10, 4 );
	}

	/**
	 * Whether a value is a site's id in ReputeCast (a UUID).
	 *
	 * @param string $id The value.
	 */
	public static function is_site_id( string $id ): bool {
		return 1 === preg_match( '/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/', $id );
	}

	/**
	 * Whether an identify secret looks like one ReputeCast makes.
	 *
	 * @param string $secret The value.
	 */
	public static function is_secret( string $secret ): bool {
		return 1 === preg_match( '/^rcid_[A-Za-z0-9_-]{43}$/', $secret );
	}

	/** The tracked site's id, when tracking is on and the site connected; else ''. */
	public static function site(): string {
		$t = Options::tracking();
		$c = Options::connection();
		if ( ! self::is_site_id( $t['site'] ) || '' === $c['instance_url'] || '' === $c['workspace'] ) {
			return '';
		}
		return $t['site'];
	}

	/** The tracker's address on the connected ReputeCast. */
	public static function script_url(): string {
		return rtrim( (string) Options::connection()['instance_url'], '/' ) . '/track/v1/rc.js';
	}

	/** Loads the tracker on every page (async, so it never holds a page up). */
	public static function enqueue(): void {
		if ( '' === self::site() ) {
			return;
		}
		wp_register_script(
			self::HANDLE,
			self::script_url(),
			array(),
			null, // phpcs:ignore WordPress.WP.EnqueuedResourceParameters.MissingVersion -- ReputeCast serves the current tracker.
			array(
				'strategy'  => 'async',
				'in_footer' => false,
			)
		);
		wp_add_inline_script( self::HANDLE, self::STUB, 'before' );
		wp_enqueue_script( self::HANDLE );
	}

	/**
	 * The tracker's tag gets the workspace, the site and its consent mode.
	 *
	 * @param string $tag    The script tags.
	 * @param string $handle The script's handle.
	 * @param string $src    The script's address (not needed: the tag has it).
	 */
	public static function tag( $tag, $handle, $src ): string { // phpcs:ignore Generic.CodeAnalysis.UnusedFunctionParameter.FoundAfterLastUsed -- the filter's signature.
		$tag  = (string) $tag;
		$site = self::site();
		if ( self::HANDLE !== $handle || '' === $site ) {
			return $tag;
		}
		$attrs = ' data-workspace="' . esc_attr( (string) Options::connection()['workspace'] ) . '" data-site="' . esc_attr( $site ) . '"';
		if ( ! Options::tracking()['wait_consent'] ) {
			$attrs .= ' data-wait-consent="false"';
		}
		return (string) preg_replace( '/<script(?=[^>]*\ssrc=)/', '<script' . $attrs, $tag, 1 );
	}

	/**
	 * The signature of an identify call: hex HMAC-SHA256 of the lowercased
	 * address, with the site's identify secret.
	 *
	 * @param string $secret The identify secret.
	 * @param string $email  The address.
	 */
	public static function signature( string $secret, string $email ): string {
		return hash_hmac( 'sha256', strtolower( trim( $email ) ), $secret );
	}

	/**
	 * The signed-in customer, signed: null for visitors, for the site's
	 * staff, and without an identify secret.
	 *
	 * @return array{email: string, signature: string}|null
	 */
	public static function identify(): ?array {
		$secret = Options::tracking_secret();
		if ( '' === $secret || ! is_user_logged_in() || current_user_can( 'edit_posts' ) ) {
			return null;
		}
		$email = strtolower( trim( (string) wp_get_current_user()->user_email ) );
		if ( '' === $email ) {
			return null;
		}
		return array(
			'email'     => $email,
			'signature' => self::signature( $secret, $email ),
		);
	}

	/**
	 * The calls this page makes: who is signed in, the product it shows, and
	 * what was added to the cart since the last page.
	 *
	 * @return array<int, array{0: string, 1: array<string, mixed>}>
	 */
	public static function commands(): array {
		$out = array();
		$who = self::identify();
		if ( null !== $who ) {
			$out[] = array( 'identify', $who );
		}
		if ( WooCommerce::active() && function_exists( 'is_product' ) && is_product() ) {
			$id = (int) get_queried_object_id();
			if ( $id > 0 ) {
				$out[] = array( 'product', array( 'id' => (string) $id ) );
			}
		}
		foreach ( self::take_carts() as $cart ) {
			$out[] = array( 'cart', $cart );
		}
		return $out;
	}

	/** Prints the page's calls (the stub first: the tracker loads async). */
	public static function footer(): void {
		if ( '' === self::site() ) {
			return;
		}
		$commands = self::commands();
		if ( array() === $commands ) {
			return;
		}
		$js = self::STUB;
		foreach ( $commands as $c ) {
			$js .= 'reputecast(' . wp_json_encode( $c[0] ) . ',' . wp_json_encode( $c[1], JSON_HEX_TAG | JSON_HEX_AMP | JSON_UNESCAPED_SLASHES ) . ');';
		}
		wp_print_inline_script_tag( $js );
	}

	/**
	 * A product added to the cart: kept until the next page.
	 *
	 * @param string $cart_item_key The cart item.
	 * @param int    $product_id    The product (the parent of a variation).
	 * @param int    $quantity      How many.
	 * @param int    $variation_id  The variation, 0 for none.
	 */
	public static function added( $cart_item_key, $product_id, $quantity = 1, $variation_id = 0 ): void { // phpcs:ignore Generic.CodeAnalysis.UnusedFunctionParameter.FoundAfterLastUsed -- the action's signature; the product, not the variation, is what ReputeCast knows.
		$session = self::session();
		if ( '' === self::site() || null === $session || (int) $product_id <= 0 ) {
			return;
		}
		$carts   = $session->get( self::SESSION );
		$carts   = is_array( $carts ) ? $carts : array();
		$carts[] = array(
			'id'       => (string) (int) $product_id,
			'quantity' => max( 1, min( 1000, (int) $quantity ) ),
		);
		$session->set( self::SESSION, array_slice( $carts, -self::MAX_CARTS ) );
	}

	/**
	 * The add-to-carts waiting, once (they're cleared).
	 *
	 * @return array<int, array{id: string, quantity: int}>
	 */
	private static function take_carts(): array {
		$session = self::session();
		if ( null === $session ) {
			return array();
		}
		$carts = $session->get( self::SESSION );
		if ( ! is_array( $carts ) || array() === $carts ) {
			return array();
		}
		$session->set( self::SESSION, null );
		return array_values( $carts );
	}

	/** The WooCommerce session, or null. */
	private static function session(): ?object {
		if ( ! WooCommerce::active() || ! function_exists( 'WC' ) ) {
			return null;
		}
		$wc = WC();
		return is_object( $wc ) && isset( $wc->session ) && is_object( $wc->session ) ? $wc->session : null;
	}
}
