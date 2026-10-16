<?php
/**
 * WPML and Polylang, when one is active.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

defined( 'ABSPATH' ) || exit;

/**
 * The site's languages and a post's translations, from WPML or Polylang
 * (plugin 1.3.0). Without either, a site has one language.
 */
final class Languages {

	/** The language plugin: "wpml", "polylang" or "". */
	public static function plugin(): string {
		if ( defined( 'ICL_SITEPRESS_VERSION' ) ) {
			return 'wpml';
		}
		if ( function_exists( 'pll_current_language' ) ) {
			return 'polylang';
		}
		return '';
	}

	/**
	 * A language code as BCP 47 writes it: "pt-br" → "pt-BR", "zh-hans" →
	 * "zh-Hans", "en_US" → "en-US". '' when it isn't one.
	 *
	 * @param string $code A WPML, Polylang or WordPress code.
	 */
	public static function bcp47( string $code ): string {
		$parts = preg_split( '/[-_]/', trim( $code ) );
		if ( ! is_array( $parts ) || ! preg_match( '/^[A-Za-z]{2,3}$/', $parts[0] ) ) {
			return '';
		}
		$out = array( strtolower( $parts[0] ) );
		foreach ( array_slice( $parts, 1 ) as $p ) {
			if ( ! preg_match( '/^[A-Za-z0-9]{2,8}$/', $p ) ) {
				return '';
			}
			if ( 2 === strlen( $p ) ) {
				$out[] = strtoupper( $p );
			} elseif ( 4 === strlen( $p ) ) {
				$out[] = ucfirst( strtolower( $p ) );
			} else {
				$out[] = strtolower( $p );
			}
		}
		return implode( '-', $out );
	}

	/**
	 * The site's languages, for the block's language setting.
	 *
	 * @return array<int, array{code: string, name: string}>
	 */
	public static function site_languages(): array {
		$out = array();
		if ( 'wpml' === self::plugin() ) {
			$langs = apply_filters( 'wpml_active_languages', null, array( 'skip_missing' => 0 ) );
			foreach ( is_array( $langs ) ? $langs : array() as $l ) {
				$code = self::bcp47( (string) ( $l['code'] ?? '' ) );
				if ( '' !== $code ) {
					$out[] = array(
						'code' => $code,
						'name' => (string) ( $l['native_name'] ?? $l['translated_name'] ?? $code ),
					);
				}
			}
		} elseif ( 'polylang' === self::plugin() && function_exists( 'pll_languages_list' ) ) {
			$slugs = (array) pll_languages_list( array( 'fields' => 'slug' ) );
			$names = (array) pll_languages_list( array( 'fields' => 'name' ) );
			foreach ( $slugs as $i => $slug ) {
				$code = self::bcp47( (string) $slug );
				if ( '' !== $code ) {
					$out[] = array(
						'code' => $code,
						'name' => (string) ( $names[ $i ] ?? $code ),
					);
				}
			}
		}
		return $out;
	}

	/** The site's default language from WPML or Polylang, or '' without one. */
	public static function site_default(): string {
		if ( 'wpml' === self::plugin() ) {
			return self::bcp47( (string) apply_filters( 'wpml_default_language', null ) );
		}
		if ( 'polylang' === self::plugin() && function_exists( 'pll_default_language' ) ) {
			return self::bcp47( (string) pll_default_language( 'slug' ) );
		}
		return '';
	}

	/**
	 * Runs a query in the site's default language: Polylang takes a "lang"
	 * argument, WPML switches languages for the query's time.
	 *
	 * @param callable $query Gets the arguments to add to its query.
	 * @return mixed What the query returned.
	 */
	public static function in_default_language( callable $query ) {
		if ( 'polylang' === self::plugin() && function_exists( 'pll_default_language' ) ) {
			return $query( array( 'lang' => (string) pll_default_language( 'slug' ) ) );
		}
		if ( 'wpml' !== self::plugin() ) {
			return $query( array() );
		}
		$current = apply_filters( 'wpml_current_language', null );
		$default = apply_filters( 'wpml_default_language', null );
		if ( ! is_string( $default ) || '' === $default || $current === $default ) {
			return $query( array() );
		}
		do_action( 'wpml_switch_language', $default );
		try {
			return $query( array() );
		} finally {
			do_action( 'wpml_switch_language', $current );
		}
	}

	/** The language of the page being shown, or '' without a language plugin. */
	public static function current(): string {
		if ( 'wpml' === self::plugin() ) {
			return self::bcp47( (string) apply_filters( 'wpml_current_language', null ) );
		}
		if ( 'polylang' === self::plugin() ) {
			return self::bcp47( (string) pll_current_language( 'slug' ) );
		}
		return '';
	}

	/**
	 * A post's language: from WPML or Polylang, else the site's.
	 *
	 * @param int $post_id The post.
	 */
	public static function of_post( int $post_id ): string {
		if ( 'wpml' === self::plugin() ) {
			$details = apply_filters( 'wpml_post_language_details', null, $post_id );
			if ( is_array( $details ) && ! empty( $details['language_code'] ) ) {
				return self::bcp47( (string) $details['language_code'] );
			}
		} elseif ( 'polylang' === self::plugin() && function_exists( 'pll_get_post_language' ) ) {
			$slug = pll_get_post_language( $post_id, 'slug' );
			if ( is_string( $slug ) && '' !== $slug ) {
				return self::bcp47( $slug );
			}
		}
		return self::bcp47( (string) get_locale() );
	}

	/**
	 * A post's translations: language code → post ID, without the post itself.
	 *
	 * @param int $post_id The post.
	 * @return array<string, int>
	 */
	public static function translations( int $post_id ): array {
		$out = array();
		if ( 'wpml' === self::plugin() ) {
			$type = 'post_' . get_post_type( $post_id );
			$trid = apply_filters( 'wpml_element_trid', null, $post_id, $type );
			$list = $trid ? apply_filters( 'wpml_get_element_translations', null, $trid, $type ) : array();
			foreach ( is_array( $list ) ? $list : array() as $lang => $t ) {
				$id = (int) ( is_object( $t ) ? ( $t->element_id ?? 0 ) : ( $t['element_id'] ?? 0 ) );
				if ( $id && $id !== $post_id ) {
					$out[ self::bcp47( (string) $lang ) ] = $id;
				}
			}
		} elseif ( 'polylang' === self::plugin() && function_exists( 'pll_get_post_translations' ) ) {
			foreach ( (array) pll_get_post_translations( $post_id ) as $lang => $id ) {
				if ( (int) $id && (int) $id !== $post_id ) {
					$out[ self::bcp47( (string) $lang ) ] = (int) $id;
				}
			}
		}
		unset( $out[''] );
		return $out;
	}
}
