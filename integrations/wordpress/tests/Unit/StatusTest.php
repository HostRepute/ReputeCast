<?php
/**
 * The status endpoint ReputeCast's "Check now" calls.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Functions;
use ReputeCast\Options;
use ReputeCast\Status;
use ReputeCast\Tests\TestCase;

final class StatusTest extends TestCase {

	public function test_the_signature_is_the_one_reputecast_computes(): void {
		// The same vector as internal/integrations TestStatusSignature.
		$key = 'rc_live_' . str_repeat( 'ab', 32 );
		$this->assertSame( '45f9841e69974b8168b517da522ad7c914b7464255f7dfeb4612923d5d6cb199', Status::signature( $key, '1792138920' ) );
	}

	public function test_only_a_call_signed_with_the_sites_key(): void {
		$now = 1792138920;
		$sig = Status::signature( self::KEY, (string) $now );
		$this->assertTrue( Status::verify( (string) $now, $sig, self::KEY, $now ) );
		$this->assertTrue( Status::verify( (string) $now, strtoupper( $sig ), self::KEY, $now ), 'hex in capitals' );
		$this->assertFalse( Status::verify( (string) $now, Status::signature( 'rc_live_' . str_repeat( '0', 64 ), (string) $now ), self::KEY, $now ), 'another key' );
		$this->assertFalse( Status::verify( (string) $now, '', self::KEY, $now ), 'no signature' );
		$this->assertFalse( Status::verify( '', $sig, self::KEY, $now ), 'no timestamp' );
		$this->assertFalse( Status::verify( (string) $now, $sig, '', $now ), 'a disconnected site answers nobody' );
	}

	public function test_during_a_rotation_any_live_key_signs(): void {
		$now    = 1792138920;
		$header = Status::signature( 'rc_live_' . str_repeat( 'cd', 32 ), (string) $now ) . ', ' . Status::signature( self::KEY, (string) $now );
		$this->assertTrue( Status::verify( (string) $now, $header, self::KEY, $now ) );
	}

	public function test_a_call_more_than_five_minutes_off_is_refused(): void {
		$then = 1792138920;
		$sig  = Status::signature( self::KEY, (string) $then );
		$this->assertTrue( Status::verify( (string) $then, $sig, self::KEY, $then + 300 ) );
		$this->assertFalse( Status::verify( (string) $then, $sig, self::KEY, $then + 301 ), 'replayed later' );
		$this->assertFalse( Status::verify( (string) $then, $sig, self::KEY, $then - 301 ), 'from the future' );
		$this->assertFalse( Status::verify( '1792138920.5', $sig, self::KEY, $then ), 'not a Unix time' );
	}

	public function test_the_endpoint_answers_401_without_a_signature(): void {
		$this->connect_site();
		$denied = Status::permission( new \WP_REST_Request() );
		$this->assertInstanceOf( \WP_Error::class, $denied );
		$this->assertSame( 401, $denied->get_error_data()['status'] );

		$ts      = (string) time();
		$request = new \WP_REST_Request(
			array(
				'X-ReputeCast-Timestamp' => $ts,
				'X-ReputeCast-Signature' => Status::signature( self::KEY, $ts ),
			)
		);
		$this->assertTrue( Status::permission( $request ) );
	}

	public function test_the_answer_is_the_versions_and_isnt_cached(): void {
		$this->connect_site();
		$response = Status::answer();
		$this->assertSame(
			array(
				'plugin'    => REPUTECAST_VERSION,
				'wordpress' => '6.9.2',
				'php'       => PHP_MAJOR_VERSION . '.' . PHP_MINOR_VERSION,
			),
			$response->get_data()
		);
		$this->assertSame( 'no-store', $response->headers['Cache-Control'] );
		$this->assertGreaterThan( 0, Options::connection()['last_check_at'] );
	}

	public function test_the_route(): void {
		$routes = array();
		Functions\when( 'register_rest_route' )->alias(
			function ( $ns, $route, $args ) use ( &$routes ) {
				$routes[ $ns . $route ] = $args;
				return true;
			}
		);
		Status::register_routes();
		$this->assertSame( 'GET', $routes['reputecast/v1/status']['methods'] );
		$this->assertSame( array( Status::class, 'permission' ), $routes['reputecast/v1/status']['permission_callback'] );
	}
}
