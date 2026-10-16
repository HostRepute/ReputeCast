<?php
/**
 * Report in: the site's health, for ReputeCast's WordPress sites page.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

use WP_Error;

defined( 'ABSPATH' ) || exit;

/**
 * The plugin reports in when it connects, every 12 hours and with every
 * post: plugin, WordPress and PHP versions, the feed URL, the categories
 * (for the post-draft rule) and where forms are used. ReputeCast never polls
 * the site; "Check now" calls the status endpoint once.
 */
final class Reporter {

	const CRON_HOOK = 'reputecast_report';

	/** Reports in every 12 hours, from about a minute from now. */
	public static function schedule(): void {
		if ( false === wp_next_scheduled( self::CRON_HOOK ) ) {
			wp_schedule_event( time() + MINUTE_IN_SECONDS, 'twicedaily', self::CRON_HOOK );
		}
	}

	/** Stops reporting in. */
	public static function unschedule(): void {
		wp_unschedule_hook( self::CRON_HOOK );
	}

	/** The 12-hour job: report in, and refresh the forms list. */
	public static function run(): void {
		if ( ! Options::is_connected() ) {
			return;
		}
		self::report();
		Forms::refresh();
	}

	/**
	 * Reports in now, and keeps what ReputeCast answers: the workspace, the
	 * site's address as ReputeCast lists it, and the newest plugin release.
	 *
	 * @param string $language The language of ReputeCast's messages.
	 * @return array<string, mixed>|WP_Error
	 */
	public static function report( string $language = '' ) {
		$client = Client::connected( $language );
		if ( null === $client ) {
			return new WP_Error( 'reputecast_not_connected', __( 'This site isn’t connected to ReputeCast.', 'reputecast' ) );
		}
		$result = $client->report( self::payload() );
		if ( is_wp_error( $result ) ) {
			Options::update(
				array(
					'last_report_error'     => $result->get_error_message(),
					'last_report_failed_at' => time(),
				)
			);
			return $result;
		}
		$c       = Options::connection();
		$changes = array(
			'last_report_at'    => time(),
			'last_report_error' => '',
			'site'              => is_string( $result['site'] ?? null ) ? sanitize_text_field( $result['site'] ) : $c['site'],
			'workspace_name'    => is_string( $result['workspaceName'] ?? null ) ? sanitize_text_field( $result['workspaceName'] ) : $c['workspace_name'],
			'latest_plugin'     => is_string( $result['latestPlugin'] ?? null ) && preg_match( '/^\d+\.\d+(\.\d+)?$/', $result['latestPlugin'] ) ? $result['latestPlugin'] : '',
		);
		$slug    = is_string( $result['workspace'] ?? null ) ? self::slug( $result['workspace'] ) : '';
		if ( '' !== $slug ) {
			$changes['workspace'] = $slug;
		}
		Options::update( $changes );
		return $result;
	}

	/**
	 * The report, with exactly the fields ReputeCast's API takes
	 * (WordPressReportInput).
	 *
	 * @return array<string, mixed>
	 */
	public static function payload(): array {
		return array(
			'siteUrl'          => Form_Usage::clip( home_url(), 300 ),
			'name'             => Form_Usage::clip( html_entity_decode( wp_strip_all_tags( (string) get_bloginfo( 'name' ) ), ENT_QUOTES | ENT_HTML5, 'UTF-8' ), 200 ),
			'pluginVersion'    => REPUTECAST_VERSION,
			'wordpressVersion' => Form_Usage::clip( (string) get_bloginfo( 'version' ), 20 ),
			'phpVersion'       => PHP_MAJOR_VERSION . '.' . PHP_MINOR_VERSION,
			'feedUrl'          => Form_Usage::clip( (string) get_feed_link(), 500 ),
			'languagePlugin'   => Languages::plugin(),
			'categories'       => self::categories(),
			'forms'            => Form_Usage::scan(),
		);
	}

	/**
	 * The post categories, for the post-draft rule: in the site's default
	 * language when WPML or Polylang translates them.
	 *
	 * @return array<int, array{id: string, name: string, slug: string}>
	 */
	public static function categories(): array {
		$terms = Languages::in_default_language(
			static function ( array $lang_args ) {
				return get_terms(
					array(
						'taxonomy'   => 'category',
						'hide_empty' => false,
						'number'     => 500,
						'orderby'    => 'name',
					) + $lang_args
				);
			}
		);
		$out   = array();
		foreach ( is_array( $terms ) ? $terms : array() as $term ) {
			if ( $term instanceof \WP_Term ) {
				$out[] = array(
					'id'   => (string) $term->term_id,
					'name' => Form_Usage::clip( html_entity_decode( $term->name, ENT_QUOTES | ENT_HTML5, 'UTF-8' ), 200 ),
					'slug' => Form_Usage::clip( rawurldecode( $term->slug ), 200 ),
				);
			}
		}
		return $out;
	}

	/**
	 * A workspace slug as ReputeCast writes it, or ''.
	 *
	 * @param string $slug What ReputeCast answered.
	 */
	public static function slug( string $slug ): string {
		return 1 === preg_match( '/^[a-z0-9][a-z0-9-]{0,99}$/', $slug ) ? $slug : '';
	}
}
