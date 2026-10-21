<?php
/**
 * WooCommerce: the marketing consent at the checkout, abandoned checkouts and
 * the click token kept on each order, for ReputeCast's WooCommerce stores.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

use WP_Error;

defined( 'ABSPATH' ) || exit;

/**
 * WooCommerce has no marketing consent and no abandoned checkouts of its
 * own. With WooCommerce active, the plugin:
 *
 * - adds an unchecked checkbox at the checkout, with exactly this text:
 *   "Email me about new products and offers from the store. Unsubscribe any
 *   time." Ticking it records the consent on the order (the text, the time);
 *   leaving it unticked adds nobody to any list;
 * - keeps the click token of the email a visitor came from (the "rc"
 *   parameter of the store's links) and writes it on their order, so
 *   ReputeCast credits the order to that email;
 * - reports the checkouts left without an order (the email typed, the cart,
 *   a link that restores it) to ReputeCast every 15 minutes.
 *
 * ReputeCast reads the order's consent and click token from the order's meta
 * through the store's REST API key; nothing else about customers is sent.
 */
final class WooCommerce {

	/** The checkbox's text: what the consent record keeps. */
	const CONSENT_TEXT = 'Email me about new products and offers from the store. Unsubscribe any time.';

	/** The checkbox's field (classic checkout) and its key (block checkout). */
	const FIELD       = 'reputecast_marketing';
	const BLOCK_FIELD = 'reputecast/marketing';

	/** The order meta ReputeCast reads (internal/stores/woocommerce). */
	const META_MARKETING    = '_reputecast_marketing';
	const META_CONSENT_TEXT = '_reputecast_consent_text';
	const META_CONSENT_AT   = '_reputecast_consent_at';
	const META_CLICK        = '_reputecast_click';
	const META_CHECKOUT     = '_reputecast_checkout';

	/** The click token's cookie, and how long it's kept (the longest attribution window). */
	const CLICK_COOKIE = 'reputecast_click';
	const CLICK_DAYS   = 30;

	/** The checkouts not reported yet, and the job that reports them. */
	const OPTION    = 'reputecast_checkouts';
	const CRON_HOOK = 'reputecast_checkouts';
	const SCHEDULE  = 'reputecast_quarter_hour';

	/** A checkout is abandoned an hour after its last change; one older than 7 days is let go. */
	const KEEP_DAYS = 7;
	const MAX_KEPT  = 500;
	const BATCH     = 200;

	/** The cart link's query parameter. */
	const RECOVER = 'reputecast_cart';

	/** The WooCommerce version, '' when WooCommerce isn't active. */
	public static function version(): string {
		$version = defined( 'WC_VERSION' ) ? (string) constant( 'WC_VERSION' ) : '';
		/**
		 * The WooCommerce version the plugin sees ('' when it isn't active).
		 *
		 * @param string $version WC_VERSION.
		 */
		return (string) apply_filters( 'reputecast_woocommerce_version', $version );
	}

	/** Whether WooCommerce is active. */
	public static function active(): bool {
		return '' !== self::version();
	}

	/** Whether the checkout shows the consent checkbox (on unless filtered off). */
	public static function consent_enabled(): bool {
		/**
		 * Whether the checkout shows ReputeCast's marketing consent checkbox.
		 *
		 * @param bool $enabled On by default.
		 */
		return (bool) apply_filters( 'reputecast_woocommerce_consent', true );
	}

	/** Whether abandoned checkouts are reported (on unless filtered off). */
	public static function checkouts_enabled(): bool {
		/**
		 * Whether the plugin reports abandoned checkouts to ReputeCast.
		 *
		 * @param bool $enabled On by default.
		 */
		return (bool) apply_filters( 'reputecast_woocommerce_checkouts', true );
	}

	/** The checkbox's text, as the shopper reads it. */
	public static function consent_text(): string {
		return __( 'Email me about new products and offers from the store. Unsubscribe any time.', 'reputecast' );
	}

	/**
	 * WooCommerce in the site's report (WordPressReportInput.woocommerce), or
	 * null without WooCommerce.
	 *
	 * @return array{version: string, consent: bool, checkouts: bool}|null
	 */
	public static function report(): ?array {
		if ( ! self::active() ) {
			return null;
		}
		return array(
			'version'   => Form_Usage::clip( self::version(), 20 ),
			'consent'   => self::consent_enabled(),
			'checkouts' => self::checkouts_enabled(),
		);
	}

	/** Hooks in, once WooCommerce has loaded. */
	public static function register(): void {
		add_action( 'woocommerce_init', array( self::class, 'boot' ) );
	}

	/** WooCommerce is there: the checkbox, the click token and the checkouts. */
	public static function boot(): void {
		if ( ! self::active() ) {
			return;
		}
		add_action( 'template_redirect', array( self::class, 'keep_click' ), 5 );
		add_action( 'template_redirect', array( self::class, 'recover' ), 6 );
		if ( self::consent_enabled() ) {
			// Under the email, in Contact information, as both checkouts show it.
			add_action( 'woocommerce_after_checkout_billing_form', array( self::class, 'print_checkbox' ) );
			if ( function_exists( 'woocommerce_register_additional_checkout_field' ) ) {
				woocommerce_register_additional_checkout_field(
					array(
						'id'       => self::BLOCK_FIELD,
						'label'    => self::consent_text(),
						'location' => 'contact',
						'type'     => 'checkbox',
						'required' => false,
					)
				);
			}
		}
		add_action( 'woocommerce_checkout_create_order', array( self::class, 'classic_order' ), 10, 2 );
		add_action( 'woocommerce_store_api_checkout_update_order_from_request', array( self::class, 'block_order' ), 10, 2 );
		if ( self::checkouts_enabled() ) {
			add_action( 'woocommerce_checkout_update_order_review', array( self::class, 'classic_review' ) );
			add_action( 'woocommerce_store_api_cart_update_customer_from_request', array( self::class, 'block_customer' ), 10, 2 );
			add_action( self::CRON_HOOK, array( self::class, 'run' ) );
			add_filter( 'cron_schedules', array( self::class, 'schedules' ) ); // phpcs:ignore WordPress.WP.CronInterval.ChangeDetected -- 15 minutes, for checkouts abandoned within the hour.
			self::schedule();
		}
	}

	/**
	 * Adds the every-15-minutes schedule.
	 *
	 * @param array<string, array<string, mixed>> $schedules WordPress's schedules.
	 * @return array<string, array<string, mixed>>
	 */
	public static function schedules( array $schedules ): array {
		$schedules[ self::SCHEDULE ] = array(
			'interval' => 15 * MINUTE_IN_SECONDS,
			'display'  => __( 'Every 15 minutes (ReputeCast checkouts)', 'reputecast' ),
		);
		return $schedules;
	}

	/** Reports checkouts every 15 minutes, once connected. */
	public static function schedule(): void {
		if ( Options::is_connected() && false === wp_next_scheduled( self::CRON_HOOK ) ) {
			wp_schedule_event( time() + MINUTE_IN_SECONDS, self::SCHEDULE, self::CRON_HOOK );
		}
	}

	/** Stops reporting checkouts. */
	public static function unschedule(): void {
		wp_unschedule_hook( self::CRON_HOOK );
	}

	/**
	 * A click token as the email's links carry it, or ''.
	 *
	 * @param mixed $raw What the address had.
	 */
	public static function click_token( $raw ): string {
		$token = is_string( $raw ) ? trim( $raw ) : '';
		return 1 === preg_match( '/^[A-Za-z0-9_.~-]{8,200}$/', $token ) ? $token : '';
	}

	/** Keeps the click token of the email a visitor came from, in a cookie. */
	public static function keep_click(): void {
		// phpcs:disable WordPress.Security.NonceVerification.Recommended -- a link from an email: read, never acted on.
		$raw = isset( $_GET['rc'] ) ? sanitize_text_field( wp_unslash( $_GET['rc'] ) ) : ( isset( $_GET['rc_click'] ) ? sanitize_text_field( wp_unslash( $_GET['rc_click'] ) ) : '' );
		// phpcs:enable WordPress.Security.NonceVerification.Recommended
		$token = self::click_token( $raw );
		if ( '' === $token || headers_sent() ) {
			return;
		}
		setcookie(
			self::CLICK_COOKIE,
			$token,
			array(
				'expires'  => time() + self::CLICK_DAYS * DAY_IN_SECONDS,
				'path'     => defined( 'COOKIEPATH' ) && '' !== COOKIEPATH ? COOKIEPATH : '/',
				'domain'   => defined( 'COOKIE_DOMAIN' ) && COOKIE_DOMAIN ? COOKIE_DOMAIN : '',
				'secure'   => is_ssl(),
				'httponly' => true,
				'samesite' => 'Lax',
			)
		);
		$_COOKIE[ self::CLICK_COOKIE ] = $token;
	}

	/** The visitor's click token, '' without one. */
	public static function current_click(): string {
		return self::click_token( isset( $_COOKIE[ self::CLICK_COOKIE ] ) ? sanitize_text_field( wp_unslash( $_COOKIE[ self::CLICK_COOKIE ] ) ) : '' );
	}

	/**
	 * The consent's order meta: the box ticked or not, its text and when.
	 *
	 * @param bool $ticked Whether the shopper ticked the box.
	 * @param int  $now    When.
	 * @return array<string, string>
	 */
	public static function consent_meta( bool $ticked, int $now ): array {
		$meta = array( self::META_MARKETING => $ticked ? 'yes' : 'no' );
		if ( $ticked ) {
			$meta[ self::META_CONSENT_TEXT ] = self::consent_text();
			$meta[ self::META_CONSENT_AT ]   = gmdate( 'Y-m-d\TH:i:s\Z', $now );
		}
		return $meta;
	}

	/** Prints the checkbox at the classic checkout, unchecked. */
	public static function print_checkbox(): void {
		printf(
			'<p class="form-row reputecast-consent"><label class="woocommerce-form__label woocommerce-form__label-for-checkbox checkbox"><input type="checkbox" class="woocommerce-form__input woocommerce-form__input-checkbox input-checkbox" name="%1$s" id="%1$s" value="1" /> <span>%2$s</span></label></p>',
			esc_attr( self::FIELD ),
			esc_html( self::consent_text() )
		);
	}

	/**
	 * The classic checkout's order: the consent, the click token, the checkout.
	 *
	 * @param \WC_Order            $order The order being created.
	 * @param array<string, mixed> $data  The checkout's posted data.
	 */
	public static function classic_order( $order, $data = array() ): void {
		unset( $data );
		// phpcs:ignore WordPress.Security.NonceVerification.Missing -- WooCommerce checked the checkout's nonce before creating the order.
		$ticked = self::consent_enabled() && ! empty( $_POST[ self::FIELD ] );
		self::write_order( $order, $ticked );
	}

	/**
	 * The block checkout's order: the additional field's value.
	 *
	 * @param \WC_Order        $order   The order being placed.
	 * @param \WP_REST_Request $request The Store API request.
	 */
	public static function block_order( $order, $request ): void {
		$fields = is_object( $request ) && method_exists( $request, 'get_param' ) ? $request->get_param( 'additional_fields' ) : null;
		$ticked = self::consent_enabled() && is_array( $fields ) && ! empty( $fields[ self::BLOCK_FIELD ] );
		self::write_order( $order, $ticked );
	}

	/**
	 * Writes the consent, the click token and the checkout's token on an order.
	 *
	 * @param \WC_Order $order  The order.
	 * @param bool      $ticked Whether the box was ticked.
	 */
	private static function write_order( $order, bool $ticked ): void {
		if ( ! is_object( $order ) || ! method_exists( $order, 'update_meta_data' ) ) {
			return;
		}
		foreach ( self::consent_meta( $ticked, time() ) as $key => $value ) {
			$order->update_meta_data( $key, $value );
		}
		$click = self::current_click();
		if ( '' !== $click ) {
			$order->update_meta_data( self::META_CLICK, $click );
		}
		$token = self::session_checkout();
		if ( '' !== $token ) {
			$order->update_meta_data( self::META_CHECKOUT, $token );
			self::complete( $token, time() );
		}
	}

	// ── Abandoned checkouts ──────────────────────────────────────────────

	/**
	 * The classic checkout's review: the email and names typed so far.
	 *
	 * @param string $posted The form, URL-encoded.
	 */
	public static function classic_review( $posted ): void {
		$form = array();
		wp_parse_str( is_string( $posted ) ? $posted : '', $form );
		self::track(
			array(
				'email'     => isset( $form['billing_email'] ) ? (string) $form['billing_email'] : '',
				'firstName' => isset( $form['billing_first_name'] ) ? (string) $form['billing_first_name'] : '',
				'lastName'  => isset( $form['billing_last_name'] ) ? (string) $form['billing_last_name'] : '',
				'accepted'  => ! empty( $form[ self::FIELD ] ),
			)
		);
	}

	/**
	 * The block checkout's customer update: the email typed so far.
	 *
	 * @param \WC_Customer     $customer The customer.
	 * @param \WP_REST_Request $request  The Store API request.
	 */
	public static function block_customer( $customer, $request ): void {
		if ( ! is_object( $customer ) || ! method_exists( $customer, 'get_billing_email' ) ) {
			return;
		}
		$fields = is_object( $request ) && method_exists( $request, 'get_param' ) ? $request->get_param( 'additional_fields' ) : null;
		self::track(
			array(
				'email'     => (string) $customer->get_billing_email(),
				'firstName' => (string) $customer->get_billing_first_name(),
				'lastName'  => (string) $customer->get_billing_last_name(),
				'accepted'  => is_array( $fields ) && ! empty( $fields[ self::BLOCK_FIELD ] ),
			)
		);
	}

	/**
	 * Keeps the visitor's checkout: who (once they typed an email) and the
	 * cart, under the token in their session.
	 *
	 * @param array{email: string, firstName: string, lastName: string, accepted: bool} $who What the checkout says.
	 */
	public static function track( array $who ): void {
		$cart = function_exists( 'WC' ) && WC()->cart ? WC()->cart : null;
		if ( null === $cart || $cart->is_empty() ) {
			return;
		}
		$items = array();
		foreach ( $cart->get_cart() as $line ) {
			$product = $line['data'] ?? null;
			if ( ! is_object( $product ) ) {
				continue;
			}
			$image   = function_exists( 'wp_get_attachment_image_url' ) ? (string) wp_get_attachment_image_url( (int) $product->get_image_id(), 'woocommerce_thumbnail' ) : '';
			$items[] = array(
				'productId' => (string) ( $line['product_id'] ?? '' ),
				'variantId' => empty( $line['variation_id'] ) ? '' : (string) $line['variation_id'],
				'title'     => Form_Usage::clip( wp_strip_all_tags( (string) $product->get_name() ), 300 ),
				'variant'   => Form_Usage::clip( wp_strip_all_tags( function_exists( 'wc_get_formatted_variation' ) && $product->is_type( 'variation' ) ? wc_get_formatted_variation( $product, true, false ) : '' ), 200 ),
				'quantity'  => (int) ( $line['quantity'] ?? 1 ),
				'price'     => self::amount( (float) $product->get_price() ),
				'imageUrl'  => Form_Usage::clip( $image, 1000 ),
			);
		}
		$totals = $cart->get_totals();
		self::keep(
			self::session_checkout( true ),
			$who,
			$items,
			self::amount( (float) ( $totals['subtotal'] ?? 0 ) ),
			self::amount( (float) ( $totals['total'] ?? 0 ) ),
			function_exists( 'get_woocommerce_currency' ) ? (string) get_woocommerce_currency() : '',
			time()
		);
	}

	/**
	 * Keeps a checkout under its token.
	 *
	 * @param string                           $token    The checkout's token.
	 * @param array<string, mixed>             $who      Email, names, the box ticked.
	 * @param array<int, array<string, mixed>> $items    The cart.
	 * @param string                           $subtotal The subtotal, "76.00".
	 * @param string                           $total    The total, "76.00".
	 * @param string                           $currency The store's currency.
	 * @param int                              $now      When.
	 */
	public static function keep( string $token, array $who, array $items, string $subtotal, string $total, string $currency, int $now ): void {
		if ( '' === $token || array() === $items ) {
			return;
		}
		$all      = self::all();
		$previous = $all[ $token ] ?? array();
		$email    = sanitize_email( (string) ( $who['email'] ?? '' ) );
		$accepted = ! empty( $who['accepted'] ) || ( ! empty( $previous['accepted'] ) && (string) ( $previous['email'] ?? '' ) === $email );
		$record   = array(
			'token'       => $token,
			'email'       => '' !== $email ? $email : (string) ( $previous['email'] ?? '' ),
			'firstName'   => Form_Usage::clip( sanitize_text_field( (string) ( $who['firstName'] ?? '' ) ), 100 ),
			'lastName'    => Form_Usage::clip( sanitize_text_field( (string) ( $who['lastName'] ?? '' ) ), 100 ),
			'accepted'    => $accepted,
			'consentAt'   => $accepted ? (int) ( $previous['consentAt'] ?? $now ) : 0,
			'items'       => array_slice( $items, 0, 100 ),
			'subtotal'    => $subtotal,
			'total'       => $total,
			'currency'    => strtoupper( Form_Usage::clip( $currency, 3 ) ),
			'clickToken'  => self::current_click(),
			'createdAt'   => (int) ( $previous['createdAt'] ?? $now ),
			'updatedAt'   => $now,
			'completedAt' => 0,
			'reportedAt'  => 0,
		);
		if ( '' === $record['firstName'] ) {
			$record['firstName'] = (string) ( $previous['firstName'] ?? '' );
		}
		if ( '' === $record['lastName'] ) {
			$record['lastName'] = (string) ( $previous['lastName'] ?? '' );
		}
		$all[ $token ] = $record;
		self::save( $all, $now );
	}

	/**
	 * Marks a checkout as an order.
	 *
	 * @param string $token The checkout's token.
	 * @param int    $now   When.
	 */
	public static function complete( string $token, int $now ): void {
		$all = self::all();
		if ( ! isset( $all[ $token ] ) ) {
			return;
		}
		$all[ $token ]['completedAt'] = $now;
		$all[ $token ]['reportedAt']  = 0;
		self::save( $all, $now );
		if ( function_exists( 'WC' ) && WC()->session ) {
			WC()->session->set( self::OPTION, null );
		}
	}

	/**
	 * The checkouts kept, by token.
	 *
	 * @return array<string, array<string, mixed>>
	 */
	public static function all(): array {
		$all = get_option( self::OPTION, array() );
		return is_array( $all ) ? $all : array();
	}

	/**
	 * Keeps the checkouts: the newest 500, none reported and older than 7 days.
	 *
	 * @param array<string, array<string, mixed>> $all The checkouts.
	 * @param int                                 $now When.
	 */
	private static function save( array $all, int $now ): void {
		$cutoff = $now - self::KEEP_DAYS * DAY_IN_SECONDS;
		$all    = array_filter( $all, static fn( $c ) => (int) ( $c['updatedAt'] ?? 0 ) >= $cutoff || 0 === (int) ( $c['reportedAt'] ?? 0 ) );
		uasort( $all, static fn( $a, $b ) => (int) ( $b['updatedAt'] ?? 0 ) <=> (int) ( $a['updatedAt'] ?? 0 ) );
		update_option( self::OPTION, array_slice( $all, 0, self::MAX_KEPT, true ), false );
	}

	/**
	 * The visitor's checkout token, made when asked to and there's none.
	 *
	 * @param bool $make Whether to make one.
	 */
	private static function session_checkout( bool $make = false ): string {
		$session = function_exists( 'WC' ) && WC()->session ? WC()->session : null;
		if ( null === $session ) {
			return '';
		}
		$token = (string) $session->get( self::OPTION );
		if ( '' === $token && $make ) {
			$token = bin2hex( random_bytes( 16 ) );
			$session->set( self::OPTION, $token );
		}
		return $token;
	}

	/**
	 * An amount as ReputeCast reads it: "76.00".
	 *
	 * @param float $value The amount.
	 */
	public static function amount( float $value ): string {
		return number_format( max( 0, $value ), 2, '.', '' );
	}

	/**
	 * The checkouts to report: changed since they were last reported, with an
	 * email or an order.
	 *
	 * @return array<int, array<string, mixed>> CommerceWooCheckout bodies.
	 */
	public static function pending(): array {
		$out = array();
		foreach ( self::all() as $c ) {
			if ( (int) ( $c['reportedAt'] ?? 0 ) >= (int) ( $c['updatedAt'] ?? 0 ) && ( 0 === (int) ( $c['completedAt'] ?? 0 ) || (int) $c['reportedAt'] >= (int) $c['completedAt'] ) ) {
				continue;
			}
			$body = array(
				'token'       => (string) $c['token'],
				'accepted'    => ! empty( $c['accepted'] ),
				'items'       => array_values( (array) ( $c['items'] ?? array() ) ),
				'subtotal'    => (string) ( $c['subtotal'] ?? '0.00' ),
				'total'       => (string) ( $c['total'] ?? '0.00' ),
				'currency'    => (string) ( $c['currency'] ?? '' ),
				'recoveryUrl' => self::recovery_url( (string) $c['token'] ),
				'createdAt'   => gmdate( 'Y-m-d\TH:i:s\Z', (int) $c['createdAt'] ),
				'updatedAt'   => gmdate( 'Y-m-d\TH:i:s\Z', (int) $c['updatedAt'] ),
			);
			foreach ( array( 'email', 'firstName', 'lastName', 'clickToken' ) as $field ) {
				if ( '' !== (string) ( $c[ $field ] ?? '' ) ) {
					$body[ $field ] = (string) $c[ $field ];
				}
			}
			if ( ! empty( $c['accepted'] ) ) {
				$body['consentText'] = self::consent_text();
				$body['consentAt']   = gmdate( 'Y-m-d\TH:i:s\Z', (int) ( empty( $c['consentAt'] ) ? $c['updatedAt'] : $c['consentAt'] ) );
			}
			if ( (int) ( $c['completedAt'] ?? 0 ) > 0 ) {
				$body['completedAt'] = gmdate( 'Y-m-d\TH:i:s\Z', (int) $c['completedAt'] );
			}
			$out[] = $body;
			if ( count( $out ) >= self::BATCH ) {
				break;
			}
		}
		return $out;
	}

	/** The 15-minute job. */
	public static function run(): void {
		if ( Options::is_connected() && self::checkouts_enabled() ) {
			self::send( time() );
		}
	}

	/**
	 * Reports the pending checkouts, and marks them reported.
	 *
	 * @param int $now When.
	 * @return array<string, mixed>|WP_Error|null What ReputeCast answered; null with nothing to report.
	 */
	public static function send( int $now ) {
		$batch = self::pending();
		if ( array() === $batch ) {
			return null;
		}
		$client = Client::connected();
		if ( null === $client ) {
			return new WP_Error( 'reputecast_not_connected', __( 'This site isn’t connected to ReputeCast.', 'reputecast' ) );
		}
		$result = $client->woo_checkouts( Form_Usage::clip( home_url(), 300 ), $batch );
		if ( is_wp_error( $result ) ) {
			return $result;
		}
		$all = self::all();
		foreach ( $batch as $c ) {
			if ( isset( $all[ $c['token'] ] ) ) {
				$all[ $c['token'] ]['reportedAt'] = $now;
			}
		}
		self::save( $all, $now );
		return $result;
	}

	/**
	 * The link that restores a checkout's cart: signed, so nobody can guess
	 * another shopper's.
	 *
	 * @param string $token The checkout's token.
	 */
	public static function recovery_url( string $token ): string {
		return add_query_arg(
			array(
				self::RECOVER => $token,
				'k'           => self::signature( $token ),
			),
			home_url( '/' )
		);
	}

	/**
	 * A checkout token's signature.
	 *
	 * @param string $token The checkout's token.
	 */
	public static function signature( string $token ): string {
		return substr( hash_hmac( 'sha256', 'reputecast-cart|' . $token, wp_salt( 'nonce' ) ), 0, 20 );
	}

	/** The recovery link: the cart back as it was, then the checkout. */
	public static function recover(): void {
		// phpcs:disable WordPress.Security.NonceVerification.Recommended -- a signed link from an email.
		$token = isset( $_GET[ self::RECOVER ] ) ? sanitize_key( wp_unslash( $_GET[ self::RECOVER ] ) ) : '';
		$sig   = isset( $_GET['k'] ) ? sanitize_key( wp_unslash( $_GET['k'] ) ) : '';
		// phpcs:enable WordPress.Security.NonceVerification.Recommended
		if ( '' === $token || ! hash_equals( self::signature( $token ), $sig ) ) {
			return;
		}
		$checkout = self::all()[ $token ] ?? null;
		$wc       = function_exists( 'WC' ) ? WC() : null;
		if ( ! is_array( $checkout ) || null === $wc || ! $wc->cart ) {
			return;
		}
		if ( 0 === (int) ( $checkout['completedAt'] ?? 0 ) ) {
			$wc->cart->empty_cart();
			foreach ( (array) ( $checkout['items'] ?? array() ) as $item ) {
				$wc->cart->add_to_cart( (int) $item['productId'], max( 1, (int) $item['quantity'] ), (int) ( $item['variantId'] ?? 0 ) );
			}
			if ( $wc->session ) {
				$wc->session->set( self::OPTION, $token );
			}
		}
		wp_safe_redirect( function_exists( 'wc_get_checkout_url' ) ? wc_get_checkout_url() : home_url( '/' ) );
		exit;
	}

	/** Removes what the plugin keeps for WooCommerce (uninstall). */
	public static function delete_all(): void {
		delete_option( self::OPTION );
		self::unschedule();
	}
}
