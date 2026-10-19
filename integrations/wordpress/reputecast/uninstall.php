<?php
/**
 * Deleting the plugin removes what it kept: the connection, the site's key,
 * its jobs, what ReputeCast answered for each post and the WooCommerce
 * checkouts not reported yet. Content keeps its blocks and shortcodes, and
 * orders keep their consent and click token.
 *
 * @package ReputeCast
 */

defined( 'WP_UNINSTALL_PLUGIN' ) || exit;

delete_option( 'reputecast_connection' );
delete_option( 'reputecast_key' );
wp_unschedule_hook( 'reputecast_report' );
wp_unschedule_hook( 'reputecast_send_post' );
wp_unschedule_hook( 'reputecast_checkouts' );
delete_option( 'reputecast_checkouts' );
delete_post_meta_by_key( '_reputecast_draft' );
