<?php
/**
 * The plugin's hooks.
 *
 * @package ReputeCast
 */

namespace ReputeCast;

defined( 'ABSPATH' ) || exit;

/**
 * Wires the parts to WordPress.
 */
final class Plugin {

	/** Hooks in. */
	public static function boot(): void {
		add_action( 'init', array( Forms::class, 'register' ) );
		add_action( 'rest_api_init', array( Forms::class, 'register_routes' ) );
		add_action( 'rest_api_init', array( Status::class, 'register_routes' ) );
		add_action( Reporter::CRON_HOOK, array( Reporter::class, 'run' ) );
		Posts::register();
		if ( is_admin() ) {
			Settings_Page::register();
		}
	}

	/** On activation: report in every 12 hours, once connected. */
	public static function activate(): void {
		if ( Options::is_connected() ) {
			Reporter::schedule();
		}
	}

	/** On deactivation: no more jobs. The connection stays, for when it's back. */
	public static function deactivate(): void {
		Reporter::unschedule();
		wp_unschedule_hook( Posts::HOOK );
	}
}
