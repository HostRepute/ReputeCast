<?php
/**
 * Published posts go to ReputeCast's post-draft rule.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Functions;
use ReputeCast\Options;
use ReputeCast\Posts;
use ReputeCast\Tests\TestCase;

final class PostsTest extends TestCase {

	/**
	 * The site's posts, by ID.
	 *
	 * @var array<int, \WP_Post>
	 */
	private array $posts = array();

	protected function setUp(): void {
		parent::setUp();
		$this->posts = array(
			812 => new \WP_Post(
				array(
					'ID'            => 812,
					'post_title'    => 'Scheduled rollouts &amp; <em>more</em>',
					'post_excerpt'  => '',
					'post_content'  => '<!-- wp:paragraph --><p>Pick a time, and we roll out for you. [gallery ids="1,2"]</p><!-- /wp:paragraph -->',
					'post_date_gmt' => '2026-10-15 10:04:00',
				)
			),
		);
		Functions\when( 'get_post' )->alias( fn( $id ) => $this->posts[ (int) $id ] ?? null );
		Functions\when( 'get_the_title' )->alias( static fn( $p ) => $p->post_title );
		Functions\when( 'get_permalink' )->alias( static fn( $p ) => 'https://blog.acme.io/?p=' . $p->ID );
		Functions\when( 'get_the_post_thumbnail_url' )->alias( static fn( $p ) => 812 === $p->ID ? 'https://blog.acme.io/wp-content/uploads/2026/10/rollouts.png' : false );
		Functions\when( 'wp_get_post_categories' )->justReturn( array( 'announcements', '%c3%a9v%c3%a9nements' ) );
		Functions\when( 'get_post_time' )->alias( static fn( $format, $gmt, $p ) => gmdate( $format, strtotime( $p->post_date_gmt . ' UTC' ) ) );
		Functions\when( 'excerpt_remove_blocks' )->alias( static fn( $c ) => $c );
		Functions\when( 'strip_shortcodes' )->alias( static fn( $c ) => (string) preg_replace( '/\[[^\]]+\]/', '', $c ) );
		Functions\when( 'wp_trim_words' )->alias(
			static function ( $text, $n, $more ) {
				$words = preg_split( '/\s+/', trim( strip_tags( $text ) ) );
				return count( $words ) > $n ? implode( ' ', array_slice( $words, 0, $n ) ) . $more : implode( ' ', $words );
			}
		);
	}

	public function test_a_post_is_sent_once_it_is_published(): void {
		$this->connect_site();
		$post = $this->posts[812];
		Posts::transition( 'publish', 'draft', $post );
		$this->assertSame( array( 812 ), $this->scheduled[0]['args'] );
		$this->assertSame( Posts::HOOK, $this->scheduled[0]['hook'] );
		$this->assertSame( array(), $this->requests, 'publishing never waits for ReputeCast' );

		// Updating a published post, a page, a password-protected post: nothing.
		Posts::transition( 'publish', 'publish', $post );
		Posts::transition( 'publish', 'draft', new \WP_Post( array( 'ID' => 3, 'post_type' => 'page' ) ) );
		Posts::transition( 'publish', 'future', new \WP_Post( array( 'ID' => 4, 'post_password' => 'secret' ) ) );
		Posts::transition( 'private', 'draft', new \WP_Post( array( 'ID' => 5, 'post_status' => 'private' ) ) );
		$this->assertCount( 1, $this->scheduled );
	}

	public function test_a_scheduled_post_counts_when_it_goes_live(): void {
		$this->connect_site();
		Posts::transition( 'publish', 'future', $this->posts[812] );
		$this->assertCount( 1, $this->scheduled );
	}

	public function test_a_disconnected_site_sends_nothing(): void {
		Posts::transition( 'publish', 'draft', $this->posts[812] );
		$this->assertSame( array(), $this->scheduled );
		Posts::send( 812 );
		$this->assertSame( array(), $this->requests );
	}

	public function test_the_post_as_the_api_takes_it_then_a_report(): void {
		$this->connect_site();
		$this->stub_site();
		$this->answer(
			'create-wordpress-draft',
			200,
			array(
				'created'    => true,
				'campaignId' => '01a14383-b881-776e-a079-41690f27069d',
				'name'       => 'Scheduled rollouts & more',
			)
		);
		$this->answer_report();
		Posts::send( 812 );

		$calls = $this->calls();
		$this->assertSame( array( 'create-wordpress-draft', 'report-wordpress-site' ), array_column( $calls, 'operationId' ) );
		$this->assertSame(
			array(
				'id'          => '812',
				'url'         => 'https://blog.acme.io/?p=812',
				'title'       => 'Scheduled rollouts & more',
				'excerpt'     => 'Pick a time, and we roll out for you.',
				'imageUrl'    => 'https://blog.acme.io/wp-content/uploads/2026/10/rollouts.png',
				'categories'  => array( 'announcements', 'événements' ),
				'publishedAt' => '2026-10-15T10:04:00Z',
				'language'    => 'en-US',
			),
			$calls[0]['body']['post']
		);

		// What ReputeCast did, on the post and on the settings page.
		$this->assertTrue( $this->meta[812][ Posts::META ]['created'] );
		$recent = Options::connection()['posts'];
		$this->assertSame( '01a14383-b881-776e-a079-41690f27069d', $recent[0]['campaignId'] );
		$this->assertSame( 'Scheduled rollouts & more', $recent[0]['title'] );
	}

	public function test_a_written_excerpt_wins_and_empty_fields_are_left_out(): void {
		$this->connect_site();
		$this->stub_site();
		$this->posts[812]->post_excerpt = 'The short version.';
		Functions\when( 'get_the_post_thumbnail_url' )->justReturn( false );
		Functions\when( 'wp_get_post_categories' )->justReturn( array() );
		$payload = Posts::payload( $this->posts[812] );
		$this->assertSame( 'The short version.', $payload['excerpt'] );
		$this->assertArrayNotHasKey( 'imageUrl', $payload );
		$this->assertArrayNotHasKey( 'categories', $payload );
	}

	public function test_a_post_without_a_title_isnt_sent(): void {
		$this->connect_site();
		$this->posts[812]->post_title = '  ';
		Posts::send( 812 );
		$this->assertSame( array(), $this->requests );
		$this->assertSame( 'no_title', Options::connection()['posts'][0]['reason'] );
	}

	public function test_when_reputecast_doesnt_answer(): void {
		$this->connect_site();
		$this->stub_site();
		Posts::send( 812 );
		$recent = Options::connection()['posts'];
		$this->assertStringContainsString( 'ReputeCast didn’t answer', $recent[0]['error'] );
		$this->assertCount( 2, $this->requests, 'the draft, then the report' );
	}

	public function test_the_settings_page_lists_the_five_latest(): void {
		$this->connect_site();
		foreach ( range( 1, 7 ) as $i ) {
			Posts::remember(
				new \WP_Post(
					array(
						'ID'         => $i,
						'post_title' => "Post $i",
					)
				),
				array(
					'created' => false,
					'reason'  => 'category',
				)
			);
		}
		// Sent again: it moves to the top, once.
		Posts::remember(
			new \WP_Post(
				array(
					'ID'         => 4,
					'post_title' => 'Post 4',
				)
			),
			array( 'created' => true )
		);
		$recent = Options::connection()['posts'];
		$this->assertSame( array( 4, 7, 6, 5, 3 ), array_column( $recent, 'post' ) );
		$this->assertTrue( $recent[0]['created'] );
	}

	public function test_without_a_language_plugin_a_post_is_alone(): void {
		list( $main, $translations ) = Posts::group( $this->posts[812] );
		$this->assertSame( 812, $main->ID );
		$this->assertSame( array(), $translations );
	}
}
