<?php
/**
 * Site tracking: the tracker's tag, the page's calls and the identify
 * signature, as ReputeCast's collect endpoint reads them.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Filters;
use Brain\Monkey\Functions;
use ReputeCast\Options;
use ReputeCast\Settings_Page;
use ReputeCast\Tests\TestCase;
use ReputeCast\Tracking;

final class TrackingTest extends TestCase {

	const SITE   = '0192f0a0-0000-7000-8000-00000000c0de';
	const SECRET = 'rcid_0123456789abcdefghijklmnopqrstuvwxyzABCDEFG';

	protected function setUp(): void {
		parent::setUp();
		Functions\when( 'WC' )->justReturn(
			(object) array(
				'session' => null,
				'cart'    => null,
			)
		);
	}

	/** The contract with internal/tracker: the script's address and the signature (tracker.Signature's vector). */
	public function test_the_contract_with_reputecast(): void {
		$this->connect_site();
		Options::set_tracking( self::SITE, true );
		$this->assertSame( self::INSTANCE . '/track/v1/rc.js', Tracking::script_url() );
		$this->assertSame( '12a36b417d210c68b4e1cc84673dcc2e19be54e3454c68c9317271f6fcddf6df', Tracking::signature( self::SECRET, ' Sam.Rivera@Outlook.com ' ) );
		$this->assertTrue( Tracking::is_secret( self::SECRET ) );
		$this->assertFalse( Tracking::is_secret( 'rcid_short' ) );
	}

	public function test_the_tag_names_the_workspace_and_the_site(): void {
		$tag = '<script id="reputecast-tracker-js-before">window.reputecast=…</script>' . "\n" . '<script src="' . self::INSTANCE . '/track/v1/rc.js" id="reputecast-tracker-js" async></script>';
		$this->assertSame( $tag, Tracking::tag( $tag, Tracking::HANDLE, '' ), 'off until a site id is saved' );
		$this->connect_site();
		Options::set_tracking( self::SITE, true );
		$out = Tracking::tag( $tag, Tracking::HANDLE, '' );
		$this->assertStringContainsString( '<script data-workspace="acme-newsletter" data-site="' . self::SITE . '" src=', $out );
		$this->assertStringNotContainsString( 'data-wait-consent', $out, 'the tracker waits by default' );
		$this->assertStringStartsWith( '<script id="reputecast-tracker-js-before">', $out, 'the inline stub is left alone' );
		Options::set_tracking( self::SITE, false );
		$this->assertStringContainsString( 'data-wait-consent="false"', Tracking::tag( $tag, Tracking::HANDLE, '' ) );
		$this->assertSame( 'other', Tracking::tag( 'other', 'jquery', '' ) );
	}

	public function test_signed_in_customers_are_identified_with_the_secret(): void {
		$this->connect_site();
		Options::set_tracking( self::SITE, true );
		Functions\when( 'is_user_logged_in' )->justReturn( true );
		Functions\when( 'current_user_can' )->justReturn( false );
		Functions\when( 'wp_get_current_user' )->justReturn( (object) array( 'user_email' => 'Sam.Rivera@Outlook.com' ) );
		$this->assertNull( Tracking::identify(), 'no secret, nobody identified' );
		Options::set_tracking_secret( self::SECRET );
		$this->assertStringStartsWith( 'v1:', $this->options[ Options::TRACKING_SECRET ], 'kept encrypted' );
		$this->assertSame(
			array(
				'email'     => 'sam.rivera@outlook.com',
				'signature' => '12a36b417d210c68b4e1cc84673dcc2e19be54e3454c68c9317271f6fcddf6df',
			),
			Tracking::identify()
		);
		Functions\when( 'current_user_can' )->justReturn( true );
		$this->assertNull( Tracking::identify(), 'the site’s staff aren’t identified' );
	}

	public function test_products_and_add_to_carts(): void {
		$this->connect_site();
		Options::set_tracking( self::SITE, true );
		Filters\expectApplied( 'reputecast_woocommerce_version' )->andReturn( '10.2.1' );
		$session = new class() {
			public array $data = array();
			public function get( $k ) {
				return $this->data[ $k ] ?? null;
			}
			public function set( $k, $v ) {
				$this->data[ $k ] = $v;
			}
		};
		Functions\when( 'WC' )->justReturn(
			(object) array(
				'session' => $session,
				'cart'    => null,
			)
		);
		Functions\when( 'is_user_logged_in' )->justReturn( false );
		Functions\when( 'is_product' )->justReturn( true );
		Functions\when( 'get_queried_object_id' )->justReturn( 8841203 );
		Tracking::added( 'k1', 8841203, 2 );
		$this->assertSame(
			array(
				array( 'product', array( 'id' => '8841203' ) ),
				array(
					'cart',
					array(
						'id'       => '8841203',
						'quantity' => 2,
					),
				),
			),
			Tracking::commands()
		);
		$this->assertSame( array( array( 'product', array( 'id' => '8841203' ) ) ), Tracking::commands(), 'an add-to-cart is called once' );
	}

	public function test_the_settings(): void {
		$this->assertSame( 'error', Settings_Page::do_tracking( 'shop', true, '', false )['type'] );
		$this->assertSame( 'error', Settings_Page::do_tracking( self::SITE, true, 'not-a-secret', false )['type'] );
		$this->assertSame( 'success', Settings_Page::do_tracking( self::SITE, false, self::SECRET, false )['type'] );
		$this->assertSame(
			array(
				'site'         => self::SITE,
				'wait_consent' => false,
			),
			Options::tracking()
		);
		$this->assertSame( self::SECRET, Options::tracking_secret() );
		Settings_Page::do_tracking( '', true, '', true );
		$this->assertSame( '', Options::tracking_secret() );
		$this->assertSame( '', Tracking::site(), 'off' );
	}
}
