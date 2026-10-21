<?php
/**
 * WooCommerce: the checkout's consent, the click token on orders and the
 * abandoned-checkout report.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Filters;
use Brain\Monkey\Functions;
use ReputeCast\Reporter;
use ReputeCast\Tests\TestCase;
use ReputeCast\WooCommerce;

if ( ! defined( 'DAY_IN_SECONDS' ) ) {
	define( 'DAY_IN_SECONDS', 86400 );
}

final class WooCommerceTest extends TestCase {

	protected function setUp(): void {
		parent::setUp();
		Functions\stubs(
			array(
				'sanitize_email' => static fn( $e ) => filter_var( trim( (string) $e ), FILTER_VALIDATE_EMAIL ) ? strtolower( trim( (string) $e ) ) : '',
				'add_query_arg'  => static fn( array $args, string $url ) => $url . ( str_contains( $url, '?' ) ? '&' : '?' ) . http_build_query( $args ),
			)
		);
		// WooCommerce without a cart or a session, unless a test says otherwise.
		Functions\when( 'WC' )->justReturn(
			(object) array(
				'session' => null,
				'cart'    => null,
			)
		);
		$_COOKIE = array();
		$_POST   = array();
	}

	protected function tearDown(): void {
		$_COOKIE = array();
		$_POST   = array();
		parent::tearDown();
	}

	/** WooCommerce 10.2.1 is active. */
	private function woo(): void {
		Filters\expectApplied( 'reputecast_woocommerce_version' )->andReturn( '10.2.1' );
	}

	public function test_the_report_says_woocommerce_is_there(): void {
		$this->assertNull( WooCommerce::report(), 'no WooCommerce, nothing in the report' );
		$this->woo();
		$this->assertSame(
			array(
				'version'   => '10.2.1',
				'consent'   => true,
				'checkouts' => true,
			),
			WooCommerce::report()
		);

		$this->connect_site();
		$this->stub_site();
		$this->woo();
		$this->answer_report();
		Reporter::report();
		$call = $this->calls()[0];
		$this->assertSame( 'report-wordpress-site', $call['operationId'] );
		$this->assertSame( '10.2.1', $call['body']['woocommerce']['version'] );
		$this->assertTrue( $call['body']['woocommerce']['consent'] );
	}

	public function test_the_checkbox_is_unchecked_with_its_exact_text(): void {
		ob_start();
		WooCommerce::print_checkbox();
		$html = (string) ob_get_clean();
		$this->assertStringContainsString( 'Email me about new products and offers from the store. Unsubscribe any time.', $html );
		$this->assertStringContainsString( 'name="reputecast_marketing"', $html );
		$this->assertStringNotContainsString( 'checked', $html );
		$this->assertSame( WooCommerce::CONSENT_TEXT, WooCommerce::consent_text() );
	}

	public function test_the_order_keeps_the_consent_and_the_click_token(): void {
		$order = new class() {
			/** @var array<string, string> */
			public array $meta = array();
			public function update_meta_data( string $key, string $value ): void {
				$this->meta[ $key ] = $value;
			}
		};
		$_COOKIE['reputecast_click'] = 'c1.AbCdEf0123456789';
		$_POST['reputecast_marketing'] = '1';
		WooCommerce::classic_order( $order, array() );
		$this->assertSame( 'yes', $order->meta['_reputecast_marketing'] );
		$this->assertSame( WooCommerce::CONSENT_TEXT, $order->meta['_reputecast_consent_text'] );
		$this->assertMatchesRegularExpression( '/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/', $order->meta['_reputecast_consent_at'] );
		$this->assertSame( 'c1.AbCdEf0123456789', $order->meta['_reputecast_click'] );

		// Not ticked: no consent, and nothing about it but "no".
		$unticked = clone $order;
		$unticked->meta = array();
		$_POST          = array();
		WooCommerce::classic_order( $unticked, array() );
		$this->assertSame( 'no', $unticked->meta['_reputecast_marketing'] );
		$this->assertArrayNotHasKey( '_reputecast_consent_text', $unticked->meta );
	}

	public function test_click_tokens(): void {
		$this->assertSame( 'c1.AbCdEf0123456789', WooCommerce::click_token( ' c1.AbCdEf0123456789 ' ) );
		$this->assertSame( '', WooCommerce::click_token( 'short' ) );
		$this->assertSame( '', WooCommerce::click_token( '<script>alert(1)</script>' ) );
		$this->assertSame( '', WooCommerce::click_token( array( 'x' ) ) );
	}

	public function test_abandoned_checkouts_are_reported_once_and_again_when_they_change(): void {
		$this->connect_site();
		$now   = 1760500000;
		$items = array(
			array(
				'productId' => '8841203',
				'variantId' => '',
				'title'     => 'Acme Logo Hoodie',
				'variant'   => 'Size M',
				'quantity'  => 1,
				'price'     => '58.00',
				'imageUrl'  => '',
			),
		);
		WooCommerce::keep(
			'4f1c8e2b7d03a9e2c4f2a9e7d1b34e6f',
			array(
				'email'     => 'Grace.Liu@example.com',
				'firstName' => 'Grace',
				'lastName'  => 'Liu',
				'accepted'  => true,
			),
			$items,
			'58.00',
			'58.00',
			'usd',
			$now
		);
		WooCommerce::keep( '0a1b2c3d4e5f60718293a4b5c6d7e8f9', array( 'email' => '' ), $items, '58.00', '58.00', 'USD', $now + 60 );

		$pending = WooCommerce::pending();
		$this->assertCount( 2, $pending );
		$grace = $pending[1]['token'] === '4f1c8e2b7d03a9e2c4f2a9e7d1b34e6f' ? $pending[1] : $pending[0];
		$this->assertSame( 'grace.liu@example.com', $grace['email'] );
		$this->assertTrue( $grace['accepted'] );
		$this->assertSame( WooCommerce::CONSENT_TEXT, $grace['consentText'] );
		$this->assertSame( 'USD', $grace['currency'] );
		$this->assertStringContainsString( 'reputecast_cart=4f1c8e2b7d03a9e2c4f2a9e7d1b34e6f', $grace['recoveryUrl'] );
		$this->assertStringContainsString( 'k=' . WooCommerce::signature( '4f1c8e2b7d03a9e2c4f2a9e7d1b34e6f' ), $grace['recoveryUrl'] );

		$this->answer(
			'report-woocommerce-checkouts',
			200,
			array(
				'store'    => 'store.example.com',
				'accepted' => 2,
				'ignored'  => 0,
			)
		);
		$result = WooCommerce::send( $now + 120 );
		$this->assertSame( 2, $result['accepted'] );
		$call = $this->calls()[0];
		$this->assertSame( 'report-woocommerce-checkouts', $call['operationId'] );
		$this->assertSame( 'https://blog.acme.io', $call['body']['siteUrl'] );
		$this->assertCount( 2, $call['body']['checkouts'] );
		$this->assertSame( array(), WooCommerce::pending(), 'reported once' );

		// The order: reported again, with when it became one.
		WooCommerce::complete( '4f1c8e2b7d03a9e2c4f2a9e7d1b34e6f', $now + 3600 );
		$again = WooCommerce::pending();
		$this->assertCount( 1, $again );
		$this->assertSame( '2025-10-15T04:46:40Z', $again[0]['completedAt'] );
	}

	public function test_a_failed_report_keeps_the_checkouts(): void {
		$this->connect_site();
		WooCommerce::keep(
			'4f1c8e2b7d03a9e2c4f2a9e7d1b34e6f',
			array( 'email' => 'grace.liu@example.com' ),
			array(
				array(
					'productId' => '1',
					'title'     => 'Mug',
					'quantity'  => 1,
					'price'     => '18.00',
				),
			),
			'18.00',
			'18.00',
			'USD',
			1760500000
		);
		$this->answer(
			'report-woocommerce-checkouts',
			409,
			array(
				'title'  => 'Conflict',
				'status' => 409,
				'detail' => 'No WooCommerce store is connected at blog.acme.io.',
				'code'   => 'store_not_connected',
			)
		);
		$this->assertInstanceOf( \WP_Error::class, WooCommerce::send( 1760500100 ) );
		$this->assertCount( 1, WooCommerce::pending(), 'kept for the next run' );
	}

	public function test_old_checkouts_are_let_go(): void {
		$items = array(
			array(
				'productId' => '1',
				'title'     => 'Mug',
				'quantity'  => 1,
				'price'     => '18.00',
			),
		);
		WooCommerce::keep( 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', array( 'email' => 'a@example.com' ), $items, '18.00', '18.00', 'USD', 1760000000 );
		$all                                       = WooCommerce::all();
		$all['aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa']['reportedAt'] = 1760000100;
		$this->options[ WooCommerce::OPTION ]      = $all;
		WooCommerce::keep( 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', array( 'email' => 'b@example.com' ), $items, '18.00', '18.00', 'USD', 1760000000 + 8 * DAY_IN_SECONDS );
		$this->assertSame( array( 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb' ), array_keys( WooCommerce::all() ), 'a reported checkout older than 7 days is gone' );
	}
}
