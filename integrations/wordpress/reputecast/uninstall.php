<?php
/**
 * Deleting the plugin removes what it kept: the connection, the site's key,
 * its jobs and what ReputeCast answered for each post. Content keeps its
 * blocks and shortcodes.
 *
 * @package ReputeCast
 */

defined( 'WP_UNINSTALL_PLUGIN' ) || exit;

delete_option( 'reputecast_connection' );
delete_option( 'reputecast_key' );
wp_unschedule_hook( 'reputecast_report' );
wp_unschedule_hook( 'reputecast_send_post' );
delete_post_meta_by_key( '_reputecast_draft' );
