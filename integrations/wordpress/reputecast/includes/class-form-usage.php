<?php
/**
 * Where the site uses ReputeCast's signup forms.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

defined( 'ABSPATH' ) || exit;

/**
 * Finds the signup-form blocks and [reputecast_form] shortcodes in published
 * posts, pages, templates and widget areas, for the report: ReputeCast shows
 * each form's places and whether this site may submit it.
 */
final class Form_Usage {

	const MAX_POSTS  = 500;
	const MAX_FORMS  = 100;
	const MAX_PLACES = 100;

	/** The block's comment delimiter, with its attributes (JSON never holds "-->": WordPress escapes "--"). */
	const BLOCK_PATTERN = '#<!--\s+wp:reputecast/signup-form\s+(\{.*?\}\s+)?/?-->#s';

	/** The shortcode, with "[[…]]" for one shown as text. */
	const SHORTCODE_PATTERN = '/(\[?)\[reputecast_form(?![\w-])([^\]]*)\](\]?)/';

	/**
	 * The forms used, as the report sends them:
	 * [{ form, kind: block|shortcode|widget, places: [{ title, url }] }].
	 *
	 * @return array<int, array<string, mixed>>
	 */
	public static function scan(): array {
		$uses = array();
		foreach ( self::posts() as $post ) {
			$place = array( 'title' => self::clip( self::text( get_the_title( $post ) ), 200 ) );
			if ( in_array( $post->post_type, array( 'wp_template', 'wp_template_part' ), true ) ) {
				$place['url'] = self::clip( home_url( '/' ), 500 );
			} elseif ( 'wp_block' !== $post->post_type ) {
				$place['url'] = self::clip( (string) get_permalink( $post ), 500 );
			}
			foreach ( self::in_content( (string) $post->post_content ) as $use ) {
				self::add( $uses, $use['form'], $use['kind'], $place );
			}
		}
		foreach ( self::widget_areas() as $area ) {
			foreach ( self::in_content( $area['content'] ) as $use ) {
				self::add(
					$uses,
					$use['form'],
					'widget',
					array(
						'title' => self::clip( self::text( $area['name'] ), 200 ),
						'url'   => self::clip( home_url( '/' ), 500 ),
					)
				);
			}
		}
		return array_slice( array_values( $uses ), 0, self::MAX_FORMS );
	}

	/**
	 * The forms a piece of content uses: blocks (inner blocks included), and
	 * shortcodes.
	 *
	 * @param string $content Post content or widget markup.
	 * @return array<int, array{form: string, kind: string}>
	 */
	public static function in_content( string $content ): array {
		$out = array();
		if ( str_contains( $content, 'wp:reputecast/signup-form' ) && preg_match_all( self::BLOCK_PATTERN, $content, $blocks, PREG_SET_ORDER ) ) {
			foreach ( $blocks as $block ) {
				$attrs = isset( $block[1] ) ? json_decode( trim( $block[1] ), true ) : null;
				$form  = Forms::clean_form_id( is_array( $attrs ) && is_string( $attrs['formId'] ?? null ) ? $attrs['formId'] : '' );
				if ( '' !== $form ) {
					$out[] = array(
						'form' => $form,
						'kind' => 'block',
					);
				}
			}
		}
		if ( str_contains( $content, '[reputecast_form' ) && preg_match_all( self::SHORTCODE_PATTERN, $content, $codes, PREG_SET_ORDER ) ) {
			foreach ( $codes as $code ) {
				if ( '[' === $code[1] && ']' === $code[3] ) {
					continue; // [[reputecast_form …]] shows the shortcode as text.
				}
				$form = Forms::clean_form_id( self::id_attribute( $code[2] ) );
				if ( '' !== $form ) {
					$out[] = array(
						'form' => $form,
						'kind' => 'shortcode',
					);
				}
			}
		}
		return $out;
	}

	/**
	 * The id="…" of a shortcode's attributes: quoted with " or ', or bare.
	 *
	 * @param string $atts The text after the shortcode's name.
	 */
	public static function id_attribute( string $atts ): string {
		if ( preg_match( '/(?:^|\s)id\s*=\s*(?:"([^"]*)"|\'([^\']*)\'|([^\s"\']+))/i', $atts, $m ) ) {
			return trim( $m[1] . ( $m[2] ?? '' ) . ( $m[3] ?? '' ) );
		}
		return '';
	}

	/**
	 * Adds a place to a form's use.
	 *
	 * @param array<string, array<string, mixed>> $uses  The uses so far.
	 * @param string                              $form  The form.
	 * @param string                              $kind  block, shortcode or widget.
	 * @param array<string, string>               $place Where: a title, and a URL when it has one.
	 */
	private static function add( array &$uses, string $form, string $kind, array $place ): void {
		$key = $form . '|' . $kind;
		if ( ! isset( $uses[ $key ] ) ) {
			$uses[ $key ] = array(
				'form'   => $form,
				'kind'   => $kind,
				'places' => array(),
			);
		}
		if ( in_array( $place, $uses[ $key ]['places'], true ) ) {
			return;
		}
		if ( count( $uses[ $key ]['places'] ) < self::MAX_PLACES ) {
			$uses[ $key ]['places'][] = $place;
		}
	}

	/**
	 * Published posts, pages, templates and synced patterns that may hold a form.
	 *
	 * @return array<int, \WP_Post>
	 */
	private static function posts(): array {
		global $wpdb;
		$block     = '%' . $wpdb->esc_like( 'wp:reputecast/signup-form' ) . '%';
		$shortcode = '%' . $wpdb->esc_like( '[reputecast_form' ) . '%';
		// phpcs:ignore WordPress.DB.DirectDatabaseQuery.DirectQuery, WordPress.DB.DirectDatabaseQuery.NoCaching -- once per report, every 12 hours.
		$ids = $wpdb->get_col(
			$wpdb->prepare(
				"SELECT ID FROM {$wpdb->posts} WHERE post_status = 'publish' AND post_type NOT IN ('revision', 'nav_menu_item', 'customize_changeset') AND (post_content LIKE %s OR post_content LIKE %s) ORDER BY ID LIMIT %d",
				$block,
				$shortcode,
				self::MAX_POSTS
			)
		);
		$out = array();
		foreach ( (array) $ids as $id ) {
			$post = get_post( (int) $id );
			if ( ! $post instanceof \WP_Post || post_password_required( $post ) ) {
				continue;
			}
			if ( is_post_type_viewable( $post->post_type ) || in_array( $post->post_type, array( 'wp_template', 'wp_template_part', 'wp_block' ), true ) ) {
				$out[] = $post;
			}
		}
		return $out;
	}

	/**
	 * The widget areas' content: block widgets and text widgets, by area.
	 *
	 * @return array<int, array{name: string, content: string}>
	 */
	private static function widget_areas(): array {
		global $wp_registered_sidebars;
		$sidebars = wp_get_sidebars_widgets();
		$blocks   = get_option( 'widget_block', array() );
		$texts    = get_option( 'widget_text', array() );
		$out      = array();
		foreach ( is_array( $sidebars ) ? $sidebars : array() as $sidebar => $widgets ) {
			if ( 'wp_inactive_widgets' === $sidebar || ! is_array( $widgets ) ) {
				continue;
			}
			$content = '';
			foreach ( $widgets as $widget ) {
				if ( preg_match( '/^block-(\d+)$/', (string) $widget, $m ) && is_array( $blocks ) && isset( $blocks[ $m[1] ]['content'] ) ) {
					$content .= (string) $blocks[ $m[1] ]['content'];
				} elseif ( preg_match( '/^text-(\d+)$/', (string) $widget, $m ) && is_array( $texts ) && isset( $texts[ $m[1] ]['text'] ) ) {
					$content .= (string) $texts[ $m[1] ]['text'];
				}
			}
			if ( '' !== $content ) {
				$name  = is_array( $wp_registered_sidebars ) && isset( $wp_registered_sidebars[ $sidebar ]['name'] ) ? (string) $wp_registered_sidebars[ $sidebar ]['name'] : (string) $sidebar;
				$out[] = array(
					'name'    => $name,
					'content' => $content,
				);
			}
		}
		return $out;
	}

	/**
	 * Plain text from a title: no tags, entities decoded.
	 *
	 * @param string $s A title as WordPress shows it.
	 */
	private static function text( string $s ): string {
		return html_entity_decode( wp_strip_all_tags( $s ), ENT_QUOTES | ENT_HTML5, 'UTF-8' );
	}

	/**
	 * Cuts text to a length, in characters.
	 *
	 * @param string $s   Text.
	 * @param int    $max Most characters.
	 */
	public static function clip( string $s, int $max ): string {
		return mb_substr( trim( $s ), 0, $max );
	}
}
