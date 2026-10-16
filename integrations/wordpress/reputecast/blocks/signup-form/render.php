<?php
/**
 * The signup-form block on the page: the form's container, filled by the
 * form's own script.
 *
 * @package ReputeCast
 *
 * @var array<string, mixed> $attributes The block's attributes.
 */

defined( 'ABSPATH' ) || exit;

$reputecast_container = ReputeCast\Forms::prepare(
	is_string( $attributes['formId'] ?? null ) ? $attributes['formId'] : '',
	is_string( $attributes['language'] ?? null ) ? $attributes['language'] : ''
);

if ( ! empty( $reputecast_container ) ) {
	// get_block_wrapper_attributes() escapes every attribute.
	echo '<div ' . get_block_wrapper_attributes( $reputecast_container ) . '></div>'; // phpcs:ignore WordPress.Security.EscapeOutput.OutputNotEscaped
}
