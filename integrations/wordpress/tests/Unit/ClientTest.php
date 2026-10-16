<?php
/**
 * The calls to ReputeCast.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use ReputeCast\Client;
use ReputeCast\Tests\TestCase;

final class ClientTest extends TestCase {

	public function test_every_call_carries_the_key_and_names_the_plugin(): void {
		$this->answer(
			'get-integration',
			200,
			array(
				'workspace'     => 'acme-newsletter',
				'workspaceName' => 'Acme Newsletter',
				'integration'   => 'wordpress',
				'key'           => 'WordPress · blog.acme.io',
				'triggers'      => array(),
				'actions'       => array(),
				'permissions'   => array( 'subscribers:view', 'campaigns:edit' ),
			)
		);
		$me = ( new Client( 'news.acme.io/', self::KEY, '', 'vi' ) )->me();
		$this->assertSame( 'acme-newsletter', $me['workspace'] );

		$call = $this->calls()[0];
		$this->assertSame( 'get-integration', $call['operationId'] );
		$this->assertSame( 'https://news.acme.io/api/v1/integration', $call['url'] );
		$this->assertSame( 'Bearer ' . self::KEY, $call['args']['headers']['Authorization'] );
		$this->assertSame( 'vi', $call['args']['headers']['Accept-Language'] );
		// ReputeCast's call log reads it as "WordPress, ReputeCast plugin 1.3.0".
		$this->assertSame( 'ReputeCast-WordPress/' . REPUTECAST_VERSION, $call['args']['user-agent'] );
		// The key goes only where it was pasted, never to a redirect.
		$this->assertSame( 0, $call['args']['redirection'] );
		$this->assertArrayNotHasKey( 'body', $call['args'] );
	}

	public function test_the_paths_are_the_workspaces(): void {
		$this->stub_site();
		$this->answer( 'report-wordpress-site', 200, array( 'workspace' => 'acme-newsletter', 'workspaceName' => 'Acme Newsletter', 'site' => 'blog.acme.io', 'latestPlugin' => '1.3.0' ) );
		$this->answer( 'list-wordpress-forms', 200, array( 'items' => array() ) );
		$this->answer( 'create-wordpress-draft', 200, array( 'created' => false, 'reason' => 'no_rule' ) );
		$client = new Client( self::INSTANCE, self::KEY, 'acme-newsletter' );
		$client->report( array( 'siteUrl' => 'https://blog.acme.io', 'pluginVersion' => REPUTECAST_VERSION ) );
		$client->forms();
		$client->draft( array( 'id' => '812', 'title' => 'Scheduled rollouts are here' ) );

		$calls = $this->calls();
		$this->assertSame( array( 'report-wordpress-site', 'list-wordpress-forms', 'create-wordpress-draft' ), array_column( $calls, 'operationId' ) );
		$this->assertSame( 'https://news.acme.io/api/v1/workspaces/acme-newsletter/wordpress/report', $calls[0]['url'] );
		$this->assertSame( 'application/json', $calls[0]['args']['headers']['Content-Type'] );
		$this->assertSame( array( 'post' => array( 'id' => '812', 'title' => 'Scheduled rollouts are here' ) ), $calls[2]['body'] );
	}

	public function test_a_problem_becomes_its_own_sentence(): void {
		$this->answer(
			'get-integration',
			401,
			array(
				'title'  => 'Unauthorized',
				'status' => 401,
				'detail' => 'This key was revoked on 14 Oct 2026. Connect the site again to get a new key.',
				'code'   => 'key_revoked',
			)
		);
		$err = ( new Client( self::INSTANCE, self::KEY ) )->me();
		$this->assertInstanceOf( \WP_Error::class, $err );
		$this->assertSame( 'reputecast_key_revoked', $err->get_error_code() );
		$this->assertSame( 'This key was revoked on 14 Oct 2026. Connect the site again to get a new key.', $err->get_error_message() );
		$this->assertSame( 401, $err->get_error_data()['status'] );
	}

	public function test_an_answer_that_isnt_a_problem(): void {
		$this->answer_raw( 502, '<html>Bad gateway</html>' );
		$err = ( new Client( self::INSTANCE, self::KEY ) )->me();
		$this->assertSame( 'reputecast_http_502', $err->get_error_code() );
		$this->assertSame( 'ReputeCast answered HTTP 502.', $err->get_error_message() );
	}

	public function test_a_redirect_is_refused(): void {
		$this->answer_raw( 301 );
		$err = ( new Client( 'http://news.acme.io', self::KEY ) )->me();
		$this->assertSame( 'reputecast_redirect', $err->get_error_code() );
		$this->assertStringContainsString( 'HTTP 301', $err->get_error_message() );
	}

	public function test_an_instance_that_doesnt_answer(): void {
		$err = ( new Client( self::INSTANCE, self::KEY ) )->me();
		$this->assertSame( 'reputecast_unreachable', $err->get_error_code() );
		$this->assertSame( 'ReputeCast didn’t answer at https://news.acme.io: cURL error 7: Failed to connect', $err->get_error_message() );
	}

	public function test_no_address_no_call(): void {
		$err = ( new Client( '', self::KEY ) )->me();
		$this->assertSame( 'reputecast_no_instance', $err->get_error_code() );
		$this->assertSame( array(), $this->requests );
	}

	/**
	 * @dataProvider addresses
	 */
	public function test_the_address_as_pasted( string $pasted, string $want ): void {
		$this->assertSame( $want, Client::normalize_url( $pasted ) );
	}

	public static function addresses(): array {
		return array(
			'as shown'          => array( 'https://news.acme.io', 'https://news.acme.io' ),
			'no scheme'         => array( 'news.acme.io', 'https://news.acme.io' ),
			'trailing slash'    => array( 'https://news.acme.io/', 'https://news.acme.io' ),
			'the API path'      => array( 'https://news.acme.io/api/v1/', 'https://news.acme.io' ),
			'a sub-path'        => array( 'https://acme.io/reputecast', 'https://acme.io/reputecast' ),
			'http, on purpose'  => array( 'http://10.0.4.12:8080', 'http://10.0.4.12:8080' ),
			'spaces'            => array( '  https://news.acme.io  ', 'https://news.acme.io' ),
			'nothing'           => array( '', '' ),
			'not a web address' => array( 'javascript:alert(1)', '' ),
		);
	}

	public function test_what_a_key_looks_like(): void {
		$this->assertTrue( Client::looks_like_key( self::KEY ) );
		$this->assertFalse( Client::looks_like_key( 'rc_live_5d1e…7b20' ), 'the masked key' );
		$this->assertFalse( Client::looks_like_key( 'whsec_' . str_repeat( 'a', 32 ) ) );
		$this->assertFalse( Client::looks_like_key( '' ) );
	}
}
