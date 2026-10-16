<?php
/**
 * Where the site uses its forms.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Functions;
use ReputeCast\Form_Usage;
use ReputeCast\Tests\TestCase;

final class FormUsageTest extends TestCase {

	public function test_blocks_and_shortcodes_in_content(): void {
		$content = implode(
			"\n",
			array(
				'<!-- wp:paragraph --><p>Join us.</p><!-- /wp:paragraph -->',
				'<!-- wp:reputecast/signup-form {"formId":"frm_8c2e1a"} /-->',
				'<!-- wp:group --><div class="wp-block-group"><!-- wp:reputecast/signup-form {"formId":"frm_51b0d2","language":"es"} /--></div><!-- /wp:group -->',
				'<!-- wp:reputecast/signup-form /-->',
				'<!-- wp:reputecast/signup-form {"formId":"not-a-form"} /-->',
				'<!-- wp:shortcode -->[reputecast_form id="frm_77aa01"]<!-- /wp:shortcode -->',
				"<p>[reputecast_form id='frm_77aa02'] and [reputecast_form id=frm_77aa03 /]</p>",
				'<p>Shown as text: [[reputecast_form id="frm_ffff00"]]</p>',
				'<p>[reputecast_form_other id="frm_ffff01"] [reputecast_form]</p>',
			)
		);
		$this->assertSame(
			array(
				array(
					'form' => 'frm_8c2e1a',
					'kind' => 'block',
				),
				array(
					'form' => 'frm_51b0d2',
					'kind' => 'block',
				),
				array(
					'form' => 'frm_77aa01',
					'kind' => 'shortcode',
				),
				array(
					'form' => 'frm_77aa02',
					'kind' => 'shortcode',
				),
				array(
					'form' => 'frm_77aa03',
					'kind' => 'shortcode',
				),
			),
			Form_Usage::in_content( $content )
		);
	}

	public function test_the_report_groups_places_by_form(): void {
		global $wpdb, $wp_registered_sidebars;
		$posts = array(
			3  => new \WP_Post(
				array(
					'ID'           => 3,
					'post_type'    => 'page',
					'post_title'   => 'About',
					'post_content' => '<!-- wp:reputecast/signup-form {"formId":"frm_8c2e1a"} /-->',
				)
			),
			7  => new \WP_Post(
				array(
					'ID'           => 7,
					'post_type'    => 'page',
					'post_title'   => 'Join the beta &amp; more',
					'post_content' => '[reputecast_form id="frm_51b0d2"] [reputecast_form id="frm_51b0d2"]',
				)
			),
			9  => new \WP_Post(
				array(
					'ID'           => 9,
					'post_type'    => 'wp_template_part',
					'post_title'   => 'Footer',
					'post_content' => '<!-- wp:reputecast/signup-form {"formId":"frm_8c2e1a"} /-->',
				)
			),
			11 => new \WP_Post(
				array(
					'ID'            => 11,
					'post_type'     => 'page',
					'post_title'    => 'Members only',
					'post_password' => 'secret',
					'post_content'  => '<!-- wp:reputecast/signup-form {"formId":"frm_8c2e1a"} /-->',
				)
			),
		);
		$wpdb  = new class( array_keys( $posts ) ) {
			public string $posts = 'wp_posts';
			public function __construct( private array $ids ) {}
			public function esc_like( $s ) {
				return $s;
			}
			public function prepare( $q, ...$args ) {
				return $q;
			}
			public function get_col( $q ) {
				return $this->ids;
			}
		};
		Functions\when( 'get_post' )->alias( static fn( $id ) => $posts[ $id ] ?? null );
		Functions\when( 'get_the_title' )->alias( static fn( $p ) => $p->post_title );
		Functions\when( 'get_permalink' )->alias( static fn( $p ) => 'https://blog.acme.io/' . strtolower( strtok( $p->post_title, ' ' ) ) . '/' );
		Functions\when( 'post_password_required' )->alias( static fn( $p ) => '' !== $p->post_password );
		Functions\when( 'is_post_type_viewable' )->alias( static fn( $type ) => in_array( $type, array( 'post', 'page' ), true ) );
		$wp_registered_sidebars = array( 'sidebar-1' => array( 'name' => 'Sidebar' ) );
		Functions\when( 'wp_get_sidebars_widgets' )->justReturn(
			array(
				'wp_inactive_widgets' => array( 'block-4' ),
				'sidebar-1'           => array( 'block-2', 'text-3' ),
			)
		);
		$this->options['widget_block'] = array(
			2 => array( 'content' => '<!-- wp:reputecast/signup-form {"formId":"frm_8c2e1a"} /-->' ),
			4 => array( 'content' => '<!-- wp:reputecast/signup-form {"formId":"frm_00dead"} /-->' ),
		);
		$this->options['widget_text']  = array( 3 => array( 'text' => '[reputecast_form id="frm_8c2e1a"]' ) );

		$this->assertSame(
			array(
				array(
					'form'   => 'frm_8c2e1a',
					'kind'   => 'block',
					'places' => array(
						array(
							'title' => 'About',
							'url'   => 'https://blog.acme.io/about/',
						),
						array(
							'title' => 'Footer',
							'url'   => 'https://blog.acme.io/',
						),
					),
				),
				array(
					'form'   => 'frm_51b0d2',
					'kind'   => 'shortcode',
					'places' => array(
						array(
							'title' => 'Join the beta & more',
							'url'   => 'https://blog.acme.io/join/',
						),
					),
				),
				array(
					'form'   => 'frm_8c2e1a',
					'kind'   => 'widget',
					'places' => array(
						array(
							'title' => 'Sidebar',
							'url'   => 'https://blog.acme.io/',
						),
					),
				),
			),
			Form_Usage::scan()
		);
		$wp_registered_sidebars = null;
	}

	public function test_lengths_are_cut_in_characters(): void {
		$this->assertSame( 'Café', Form_Usage::clip( '  Café au lait ', 4 ) );
	}
}
