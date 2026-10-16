<?php
/**
 * Published posts, for ReputeCast's post-draft rule.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

use WP_Post;

defined( 'ABSPATH' ) || exit;

/**
 * When a post is published, the plugin sends it to ReputeCast, which
 * decides with the site's post-draft rule whether it becomes a campaign
 * draft. The plugin doesn't decide anything, and nothing is sent to anyone
 * until a person reviews the draft in ReputeCast. Plugin 1.3.0 sends a
 * post's translations (WPML or Polylang) with it: they become the draft's
 * language versions.
 */
final class Posts {

	/** The background job that sends a published post. */
	const HOOK = 'reputecast_send_post';

	/** What ReputeCast answered for a post. */
	const META = '_reputecast_draft';

	/** How many posts the settings page lists. */
	const RECENT = 5;

	/** Hooks in. */
	public static function register(): void {
		add_action( 'transition_post_status', array( self::class, 'transition' ), 10, 3 );
		add_action( self::HOOK, array( self::class, 'send' ) );
	}

	/**
	 * A post was published: send it in the background, so publishing never
	 * waits for ReputeCast.
	 *
	 * @param string  $new_status The new status.
	 * @param string  $old_status The previous status.
	 * @param WP_Post $post       The post.
	 */
	public static function transition( $new_status, $old_status, $post ): void {
		if ( 'publish' !== $new_status || 'publish' === $old_status || ! $post instanceof WP_Post ) {
			return;
		}
		if ( ! self::sendable( $post ) || ! Options::is_connected() ) {
			return;
		}
		wp_schedule_single_event( time(), self::HOOK, array( (int) $post->ID ) );
	}

	/**
	 * A post the rule may turn into a draft: a published post, without a
	 * password.
	 *
	 * @param WP_Post $post The post.
	 */
	public static function sendable( WP_Post $post ): bool {
		return 'post' === $post->post_type && 'publish' === $post->post_status && '' === (string) $post->post_password;
	}

	/**
	 * The background job: sends the post (with its translations), then
	 * reports in.
	 *
	 * @param int $post_id The post.
	 */
	public static function send( $post_id ): void {
		$post = get_post( (int) $post_id );
		if ( ! $post instanceof WP_Post || ! self::sendable( $post ) ) {
			return;
		}
		$client = Client::connected();
		if ( null === $client ) {
			return;
		}
		list( $main, $translations ) = self::group( $post );
		$payload                     = self::payload( $main );
		if ( '' === $payload['title'] ) {
			self::remember(
				$main,
				array(
					'created' => false,
					'reason'  => 'no_title',
				)
			);
			return;
		}
		if ( array() !== $translations ) {
			$payload['translations'] = array_values( array_filter( array_map( array( self::class, 'translation' ), $translations ) ) );
		}
		$result = $client->draft( $payload );
		self::remember( $main, is_wp_error( $result ) ? array( 'error' => $result->get_error_message() ) : $result );
		Reporter::report();
	}

	/**
	 * A post and its published translations: the one in the site's default
	 * language leads (else the first published), the others are its
	 * language versions. Without WPML or Polylang, the post alone.
	 *
	 * @param WP_Post $post A published post.
	 * @return array{0: WP_Post, 1: array<int, WP_Post>}
	 */
	public static function group( WP_Post $post ): array {
		$others = Languages::translations( (int) $post->ID );
		if ( array() === $others ) {
			return array( $post, array() );
		}
		$members = array( Languages::of_post( (int) $post->ID ) => $post );
		foreach ( $others as $lang => $id ) {
			$other = get_post( $id );
			if ( $other instanceof WP_Post && self::sendable( $other ) && ! isset( $members[ $lang ] ) ) {
				$members[ $lang ] = $other;
			}
		}
		$default = Languages::site_default();
		if ( '' !== $default && isset( $members[ $default ] ) ) {
			$main = $members[ $default ];
		} else {
			$main = $post;
			foreach ( $members as $member ) {
				if ( strcmp( $member->post_date_gmt, $main->post_date_gmt ) < 0 ) {
					$main = $member;
				}
			}
		}
		$translations = array();
		foreach ( $members as $member ) {
			if ( $member->ID !== $main->ID ) {
				$translations[] = $member;
			}
		}
		return array( $main, $translations );
	}

	/**
	 * A post as ReputeCast takes it (WordPressPost).
	 *
	 * @param WP_Post $post The post.
	 * @return array<string, mixed>
	 */
	public static function payload( WP_Post $post ): array {
		$out = array(
			'id'          => (string) $post->ID,
			'url'         => Form_Usage::clip( (string) get_permalink( $post ), 1000 ),
			'title'       => Form_Usage::clip( self::text( get_the_title( $post ) ), 300 ),
			'excerpt'     => Form_Usage::clip( self::excerpt( $post ), 2000 ),
			'imageUrl'    => Form_Usage::clip( (string) get_the_post_thumbnail_url( $post, 'large' ), 1000 ),
			'categories'  => array_slice( self::categories( $post ), 0, 100 ),
			'publishedAt' => (string) get_post_time( 'Y-m-d\TH:i:s\Z', true, $post ),
			'language'    => Form_Usage::clip( Languages::of_post( (int) $post->ID ), 40 ),
		);
		return array_filter(
			$out,
			static function ( $v, $k ) {
				return in_array( $k, array( 'id', 'title' ), true ) || ( array() !== $v && '' !== $v );
			},
			ARRAY_FILTER_USE_BOTH
		);
	}

	/**
	 * A translation, for the draft's language version: its language, title,
	 * excerpt, link and featured image.
	 *
	 * @param WP_Post $post The translated post.
	 * @return array<string, string>|null
	 */
	public static function translation( WP_Post $post ): ?array {
		$p = self::payload( $post );
		if ( '' === $p['title'] || empty( $p['language'] ) ) {
			return null;
		}
		unset( $p['categories'], $p['publishedAt'] );
		return $p;
	}

	/**
	 * A post's category slugs.
	 *
	 * @param WP_Post $post The post.
	 * @return array<int, string>
	 */
	private static function categories( WP_Post $post ): array {
		$slugs = wp_get_post_categories( (int) $post->ID, array( 'fields' => 'slugs' ) );
		$out   = array();
		foreach ( is_array( $slugs ) ? $slugs : array() as $slug ) {
			$out[] = Form_Usage::clip( rawurldecode( (string) $slug ), 200 );
		}
		return $out;
	}

	/**
	 * The excerpt: the one written for the post, else its first 55 words.
	 *
	 * @param WP_Post $post The post.
	 */
	private static function excerpt( WP_Post $post ): string {
		$text = '' !== trim( (string) $post->post_excerpt ) ? (string) $post->post_excerpt : wp_trim_words( strip_shortcodes( excerpt_remove_blocks( (string) $post->post_content ) ), 55, '…' );
		return self::text( $text );
	}

	/**
	 * Plain text: no tags, entities decoded, spaces collapsed.
	 *
	 * @param string $s HTML.
	 */
	private static function text( string $s ): string {
		return trim( (string) preg_replace( '/\s+/u', ' ', html_entity_decode( wp_strip_all_tags( $s ), ENT_QUOTES | ENT_HTML5, 'UTF-8' ) ) );
	}

	/**
	 * Keeps ReputeCast's answer on the post, and in the settings page's list.
	 *
	 * @param WP_Post              $post   The post sent.
	 * @param array<string, mixed> $result The answer, or { error }.
	 */
	public static function remember( WP_Post $post, array $result ): void {
		$entry = array(
			'post'       => (int) $post->ID,
			'title'      => Form_Usage::clip( self::text( get_the_title( $post ) ), 300 ),
			'at'         => time(),
			'created'    => true === ( $result['created'] ?? false ),
			'reason'     => is_string( $result['reason'] ?? null ) ? sanitize_key( $result['reason'] ) : '',
			'campaignId' => is_string( $result['campaignId'] ?? null ) ? sanitize_text_field( $result['campaignId'] ) : '',
			'error'      => is_string( $result['error'] ?? null ) ? sanitize_text_field( $result['error'] ) : '',
		);
		update_post_meta( (int) $post->ID, self::META, $entry );
		$recent = Options::connection()['posts'];
		$recent = array_values(
			array_filter(
				is_array( $recent ) ? $recent : array(),
				static function ( $e ) use ( $entry ) {
					return is_array( $e ) && ( $e['post'] ?? 0 ) !== $entry['post'];
				}
			)
		);
		array_unshift( $recent, $entry );
		Options::update( array( 'posts' => array_slice( $recent, 0, self::RECENT ) ) );
	}
}
