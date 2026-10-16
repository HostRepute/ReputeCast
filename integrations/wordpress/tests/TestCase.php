<?php
/**
 * The plugin's unit tests: WordPress stubbed with Brain Monkey, options and
 * HTTP in memory, every API call checked against web/openapi.json.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests;

use Brain\Monkey;
use Brain\Monkey\Functions;
use Mockery\Adapter\Phpunit\MockeryPHPUnitIntegration;
use PHPUnit\Framework\TestCase as Base;
use ReputeCast\Options;

/**
 * Base test: an in-memory site at https://blog.acme.io, and a fake
 * ReputeCast that answers from a queue.
 */
abstract class TestCase extends Base {

	use MockeryPHPUnitIntegration;

	const INSTANCE = 'https://news.acme.io';
	const KEY      = 'rc_live_5d1e0c4b9a8f7e6d5c4b3a2918273645f1e2d3c4b5a69788796a5b4c3d2e7b20';

	/**
	 * The options table.
	 *
	 * @var array<string, mixed>
	 */
	protected array $options = array();

	/**
	 * The transients.
	 *
	 * @var array<string, mixed>
	 */
	protected array $transients = array();

	/**
	 * Post meta: post ID → key → value.
	 *
	 * @var array<int, array<string, mixed>>
	 */
	protected array $meta = array();

	/**
	 * The calls the plugin made: { method, url, args }.
	 *
	 * @var array<int, array<string, mixed>>
	 */
	protected array $requests = array();

	/**
	 * The answers, in order: a WordPress HTTP response or a WP_Error.
	 *
	 * @var array<int, mixed>
	 */
	protected array $answers = array();

	/**
	 * Jobs scheduled: { hook, args, recurrence }.
	 *
	 * @var array<int, array<string, mixed>>
	 */
	protected array $scheduled = array();

	protected function setUp(): void {
		parent::setUp();
		Monkey\setUp();
		Functions\stubTranslationFunctions();
		Functions\stubEscapeFunctions();
		Functions\stubs(
			array(
				'wp_json_encode'                   => static fn( $v ) => json_encode( $v, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE ),
				'wp_parse_url'                     => static fn( $url, $component = -1 ) => parse_url( $url, $component ),
				'is_wp_error'                      => static fn( $v ) => $v instanceof \WP_Error,
				'wp_strip_all_tags'                => static fn( $s ) => trim( strip_tags( (string) $s ) ),
				'sanitize_text_field'              => static fn( $s ) => trim( (string) preg_replace( '/[\r\n\t ]+/', ' ', strip_tags( (string) $s ) ) ),
				'sanitize_key'                     => static fn( $k ) => (string) preg_replace( '/[^a-z0-9_\-]/', '', strtolower( (string) $k ) ),
				'esc_url_raw'                      => static fn( $u ) => preg_match( '#^https?://[^\s<>"]+$#i', (string) $u ) ? (string) $u : '',
				'wp_unslash'                       => static fn( $v ) => is_string( $v ) ? stripslashes( $v ) : $v,
				'wp_salt'                          => 'put-your-unique-phrase-here-for-the-tests',
				'home_url'                         => static fn( $path = '' ) => 'https://blog.acme.io' . ( '' === $path ? '' : '/' . ltrim( $path, '/' ) ),
				'admin_url'                        => static fn( $path = '' ) => 'https://blog.acme.io/wp-admin/' . ltrim( $path, '/' ),
				'rest_url'                         => static fn( $path = '' ) => 'https://blog.acme.io/wp-json/' . ltrim( $path, '/' ),
				'get_bloginfo'                     => static fn( $show = '' ) => array(
					'name'    => 'Acme Blog',
					'version' => '6.9.2',
				)[ $show ] ?? '',
				'get_feed_link'                    => 'https://blog.acme.io/feed/',
				'get_locale'                       => 'en_US',
				'get_user_locale'                  => 'en_US',
				'get_current_user_id'              => 1,
				'get_option'                       => fn( $name, $fallback = false ) => array_key_exists( $name, $this->options ) ? $this->options[ $name ] : $fallback,
				'update_option'                    => function ( $name, $value ) {
					$this->options[ $name ] = $value;
					return true;
				},
				'delete_option'                    => function ( $name ) {
					unset( $this->options[ $name ] );
					return true;
				},
				'get_transient'                    => fn( $name ) => $this->transients[ $name ] ?? false,
				'set_transient'                    => function ( $name, $value ) {
					$this->transients[ $name ] = $value;
					return true;
				},
				'delete_transient'                 => function ( $name ) {
					unset( $this->transients[ $name ] );
					return true;
				},
				'update_post_meta'                 => function ( $id, $key, $value ) {
					$this->meta[ $id ][ $key ] = $value;
					return true;
				},
				'wp_remote_request'                => function ( $url, $args ) {
					$this->requests[] = array(
						'method' => $args['method'] ?? 'GET',
						'url'    => $url,
						'args'   => $args,
					);
					return array() === $this->answers ? new \WP_Error( 'http_request_failed', 'cURL error 7: Failed to connect' ) : array_shift( $this->answers );
				},
				'wp_remote_retrieve_response_code' => static fn( $r ) => is_array( $r ) ? $r['response']['code'] : '',
				'wp_remote_retrieve_body'          => static fn( $r ) => is_array( $r ) ? $r['body'] : '',
				'wp_next_scheduled'                => fn( $hook ) => array() === array_filter( $this->scheduled, static fn( $j ) => $j['hook'] === $hook ) ? false : time(),
				'wp_schedule_event'                => function ( $ts, $recurrence, $hook, $args = array() ) {
					$this->scheduled[] = compact( 'hook', 'args', 'recurrence', 'ts' );
					return true;
				},
				'wp_schedule_single_event'         => function ( $ts, $hook, $args = array() ) {
					$recurrence        = false;
					$this->scheduled[] = compact( 'hook', 'args', 'recurrence', 'ts' );
					return true;
				},
				'wp_unschedule_hook'               => function ( $hook ) {
					$this->scheduled = array_values( array_filter( $this->scheduled, static fn( $j ) => $j['hook'] !== $hook ) );
					return 1;
				},
			)
		);
	}

	protected function tearDown(): void {
		Monkey\tearDown();
		parent::tearDown();
	}

	/**
	 * Queues an answer from ReputeCast, after checking it against the
	 * operation's documented answer.
	 *
	 * @param string $operation_id The operation answered.
	 * @param int    $status       HTTP status.
	 * @param mixed  $body         The JSON answer (encoded here).
	 */
	protected function answer( string $operation_id, int $status, $body ): void {
		$json = json_encode( $body, JSON_UNESCAPED_SLASHES );
		$this->assertSame( array(), Contract::check_response( $operation_id, $status, $json ), "the mocked $status answer of $operation_id" );
		$this->answers[] = array(
			'response' => array(
				'code'    => $status,
				'message' => '',
			),
			'headers'  => array(),
			'body'     => $json,
		);
	}

	/**
	 * Queues an answer that isn't the API's (a proxy's page, a redirect).
	 *
	 * @param int    $status HTTP status.
	 * @param string $body   The body.
	 */
	protected function answer_raw( int $status, string $body = '' ): void {
		$this->answers[] = array(
			'response' => array(
				'code'    => $status,
				'message' => '',
			),
			'headers'  => array(),
			'body'     => $body,
		);
	}

	/**
	 * The calls made, each checked against web/openapi.json.
	 *
	 * @return array<int, array<string, mixed>> { method, url, args, operationId, body }.
	 */
	protected function calls(): array {
		$out = array();
		foreach ( $this->requests as $r ) {
			$body  = (string) ( $r['args']['body'] ?? '' );
			$check = Contract::check_request( $r['method'], $r['url'], $body );
			$this->assertSame( array(), $check['errors'], $r['method'] . ' ' . $r['url'] . ' against web/openapi.json' );
			$out[] = $r + array(
				'operationId' => $check['operationId'],
				'body'        => '' === $body ? null : json_decode( $body, true ),
			);
		}
		return $out;
	}

	/** A connected site, as after Connect. */
	protected function connect_site( array $values = array() ): void {
		Options::set_key( self::KEY );
		Options::update(
			$values + array(
				'instance_url'   => self::INSTANCE,
				'workspace'      => 'acme-newsletter',
				'workspace_name' => 'Acme Newsletter',
				'site'           => 'blog.acme.io',
				'connected_at'   => time() - 3600,
			)
		);
	}

	/** A report's answer. */
	protected function answer_report(): void {
		$this->answer(
			'report-wordpress-site',
			200,
			array(
				'workspace'     => 'acme-newsletter',
				'workspaceName' => 'Acme Newsletter',
				'site'          => 'blog.acme.io',
				'latestPlugin'  => '1.3.0',
			)
		);
	}

	/**
	 * Stubs the WordPress functions the report reads: no posts with forms,
	 * no widgets, no categories, unless a test says otherwise.
	 *
	 * @param array<int, \WP_Term> $terms The categories.
	 */
	protected function stub_site( array $terms = array() ): void {
		global $wpdb;
		$wpdb = new class() {
			public string $posts = 'wp_posts';
			public function esc_like( $s ) {
				return addcslashes( $s, '_%\\' );
			}
			public function prepare( $q, ...$args ) {
				return $q;
			}
			public function get_col( $q ) {
				return array();
			}
		};
		Functions\when( 'wp_get_sidebars_widgets' )->justReturn( array() );
		Functions\when( 'get_terms' )->justReturn( $terms );
	}
}
