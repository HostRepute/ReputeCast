<?php
/**
 * The block, the shortcode and the editor's list of forms.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Functions;
use ReputeCast\Forms;
use ReputeCast\Options;
use ReputeCast\Tests\TestCase;

final class FormsTest extends TestCase {

	/**
	 * Scripts enqueued: handle → { src, ver, args }.
	 *
	 * @var array<string, array<string, mixed>>
	 */
	private array $scripts = array();

	protected function setUp(): void {
		parent::setUp();
		$this->scripts = array();
		Functions\when( 'wp_enqueue_script' )->alias(
			function ( $handle, $src, $deps, $ver, $args ) {
				$this->scripts[ $handle ] = compact( 'src', 'ver', 'args' );
			}
		);
		Functions\when( 'shortcode_atts' )->alias( static fn( $pairs, $atts ) => array_merge( $pairs, array_intersect_key( $atts, $pairs ) ) );
	}

	/** The answer of GET …/wordpress/forms. */
	private function answer_forms(): void {
		$this->answer(
			'list-wordpress-forms',
			200,
			array(
				'items' => array(
					array(
						'id'          => 'frm_8c2e1a',
						'name'        => 'Newsletter signup',
						'status'      => 'live',
						'kind'        => 'embedded',
						'siteAllowed' => true,
						'shortcode'   => '[reputecast_form id="frm_8c2e1a"]',
						'script'      => 'https://news.acme.io/p/acme-newsletter/forms/frm_8c2e1a.js',
					),
					array(
						'id'          => 'frm_51b0d2',
						'name'        => 'Beta apply <b>now</b>',
						'status'      => 'paused',
						'kind'        => 'popup',
						'siteAllowed' => false,
						'shortcode'   => '[reputecast_form id="frm_51b0d2"]',
						'script'      => 'https://news.acme.io/p/acme-newsletter/forms/frm_51b0d2.js',
					),
				),
			)
		);
	}

	public function test_the_list_is_kept_with_only_what_the_plugin_uses(): void {
		$this->connect_site();
		$this->answer_forms();
		$forms = Forms::refresh();
		$this->assertSame( 'list-wordpress-forms', $this->calls()[0]['operationId'] );
		$this->assertSame(
			array(
				'id'          => 'frm_51b0d2',
				'name'        => 'Beta apply now',
				'status'      => 'paused',
				'kind'        => 'popup',
				'siteAllowed' => false,
				'script'      => 'https://news.acme.io/p/acme-newsletter/forms/frm_51b0d2.js',
			),
			$forms[1]
		);
		$this->assertSame( $forms, Options::connection()['forms'] );
		$this->assertGreaterThan( 0, Options::connection()['forms_at'] );
	}

	public function test_a_form_is_a_container_and_its_script(): void {
		$this->connect_site();
		$this->answer_forms();
		Forms::refresh();
		$this->assertSame( array( 'data-reputecast-form' => 'frm_8c2e1a' ), Forms::prepare( 'frm_8c2e1a' ) );
		$this->assertSame(
			array(
				'src'  => 'https://news.acme.io/p/acme-newsletter/forms/frm_8c2e1a.js',
				'ver'  => null,
				'args' => array(
					'strategy'  => 'async',
					'in_footer' => true,
				),
			),
			$this->scripts['reputecast-form-frm_8c2e1a']
		);
		$this->assertSame( '<div class="reputecast-form" data-reputecast-form="frm_8c2e1a"></div>', Forms::shortcode( array( 'id' => 'frm_8c2e1a' ) ) );
	}

	public function test_a_popup_form_opens_by_itself(): void {
		$this->connect_site();
		$this->answer_forms();
		Forms::refresh();
		$this->assertSame( array(), Forms::prepare( 'frm_51b0d2' ) );
		$this->assertSame( '', Forms::shortcode( array( 'id' => 'frm_51b0d2' ) ) );
		$this->assertArrayHasKey( 'reputecast-form-frm_51b0d2', $this->scripts );
	}

	public function test_forms_keep_working_after_disconnecting(): void {
		$this->connect_site();
		$this->answer_forms();
		Forms::refresh();
		$this->requests = array();
		Options::forget_key();
		$this->assertNotNull( Forms::prepare( 'frm_8c2e1a' ) );
		// A form added after the list was fetched: its script from the instance's address.
		$this->assertSame( array( 'data-reputecast-form' => 'frm_00beef' ), Forms::prepare( 'frm_00beef' ) );
		$this->assertSame( 'https://news.acme.io/p/acme-newsletter/forms/frm_00beef.js', $this->scripts['reputecast-form-frm_00beef']['src'] );
		$this->assertSame( array(), $this->requests, 'no call to ReputeCast on a page view' );
	}

	public function test_nothing_without_a_form_or_an_instance(): void {
		$this->assertNull( Forms::prepare( 'frm_8c2e1a' ), 'never connected' );
		$this->connect_site();
		$this->assertNull( Forms::prepare( '' ) );
		$this->assertNull( Forms::prepare( '"><script>' ) );
		$this->assertSame( '', Forms::shortcode( '' ) );
		$this->assertSame( array(), $this->scripts );
	}

	public function test_the_language_follows_the_block_then_the_page(): void {
		$this->connect_site();
		$this->assertSame( array( 'data-reputecast-form' => 'frm_8c2e1a', 'lang' => 'es' ), Forms::prepare( 'frm_8c2e1a', 'es' ) );
		$this->assertSame( array( 'data-reputecast-form' => 'frm_8c2e1a', 'lang' => 'pt-BR' ), Forms::prepare( 'frm_8c2e1a', 'pt-br' ) );
		// Without WPML or Polylang the page's own lang applies (LanguagePluginsTest has them).
		$this->assertSame( array( 'data-reputecast-form' => 'frm_8c2e1a' ), Forms::prepare( 'frm_8c2e1a' ) );
	}

	public function test_the_editors_list(): void {
		$this->connect_site( array( 'forms_at' => time() - 3600 ) );
		$this->answer_forms();
		Functions\when( 'current_user_can' )->alias( static fn( $cap ) => 'edit_posts' === $cap );
		$data = Forms::rest_forms( new \WP_REST_Request() )->get_data();
		$this->assertTrue( $data['connected'] );
		$this->assertSame( 'blog.acme.io', $data['site'] );
		$this->assertFalse( $data['canManage'] );
		$this->assertSame( array( 'id', 'name', 'status', 'kind', 'siteAllowed' ), array_keys( $data['forms'][0] ) );
		$this->assertSame( '', $data['error'] );
		$this->assertCount( 1, $this->requests, 'a stale list is refreshed' );

		// Fresh: no call.
		Forms::rest_forms( new \WP_REST_Request() );
		$this->assertCount( 1, $this->requests );

		// Asked to refresh, and ReputeCast doesn't answer: the kept list, and why.
		$data = Forms::rest_forms( new \WP_REST_Request( array(), array( 'refresh' => true ) ) )->get_data();
		$this->assertCount( 2, $data['forms'] );
		$this->assertStringContainsString( 'ReputeCast didn’t answer', $data['error'] );
	}

	public function test_the_editors_list_needs_edit_posts(): void {
		$routes = array();
		Functions\when( 'register_rest_route' )->alias(
			function ( $ns, $route, $args ) use ( &$routes ) {
				$routes[ $ns . $route ] = $args;
				return true;
			}
		);
		Forms::register_routes();
		$allowed = false;
		Functions\when( 'current_user_can' )->alias(
			static function ( $cap ) use ( &$allowed ) {
				return 'edit_posts' === $cap && $allowed;
			}
		);
		$this->assertFalse( ( $routes['reputecast/v1/forms']['permission_callback'] )() );
		$allowed = true;
		$this->assertTrue( ( $routes['reputecast/v1/forms']['permission_callback'] )() );
	}

	/**
	 * @dataProvider form_ids
	 */
	public function test_form_ids( string $raw, string $want ): void {
		$this->assertSame( $want, Forms::clean_form_id( $raw ) );
	}

	public static function form_ids(): array {
		return array(
			'as ReputeCast writes it' => array( 'frm_8c2e1a', 'frm_8c2e1a' ),
			'spaces'                  => array( ' frm_8c2e1a ', 'frm_8c2e1a' ),
			'no prefix'               => array( '8c2e1a', '' ),
			'markup'                  => array( 'frm_8c2e1a"><script>', '' ),
			'too long'                => array( 'frm_' . str_repeat( 'a', 37 ), '' ),
		);
	}
}
