<?php
/**
 * The few WordPress classes the plugin uses, as small stand-ins: the unit
 * tests run without WordPress (Brain Monkey stubs its functions).
 *
 * @package ReputeCast
 */

// phpcs:disable

class WP_Error {
	/** @var array<string, array<int, string>> */
	public $errors = array();
	/** @var array<string, mixed> */
	public $error_data = array();

	public function __construct( $code = '', $message = '', $data = '' ) {
		if ( '' === $code ) {
			return;
		}
		$this->errors[ $code ][] = $message;
		if ( '' !== $data ) {
			$this->error_data[ $code ] = $data;
		}
	}

	public function get_error_code() {
		$codes = array_keys( $this->errors );
		return $codes[0] ?? '';
	}

	public function get_error_message( $code = '' ) {
		$code = '' === $code ? $this->get_error_code() : $code;
		return $this->errors[ $code ][0] ?? '';
	}

	public function get_error_data( $code = '' ) {
		$code = '' === $code ? $this->get_error_code() : $code;
		return $this->error_data[ $code ] ?? null;
	}
}

class WP_Post {
	public $ID             = 0;
	public $post_type      = 'post';
	public $post_status    = 'publish';
	public $post_title     = '';
	public $post_excerpt   = '';
	public $post_content   = '';
	public $post_password  = '';
	public $post_date_gmt  = '2026-10-15 14:04:00';

	public function __construct( array $fields = array() ) {
		foreach ( $fields as $k => $v ) {
			$this->$k = $v;
		}
	}
}

class WP_Term {
	public $term_id = 0;
	public $name    = '';
	public $slug    = '';

	public function __construct( array $fields = array() ) {
		foreach ( $fields as $k => $v ) {
			$this->$k = $v;
		}
	}
}

class WP_REST_Server {
	const READABLE = 'GET';
}

class WP_REST_Request {
	/** @var array<string, string> */
	private $headers = array();
	/** @var array<string, mixed> */
	private $params = array();

	public function __construct( array $headers = array(), array $params = array() ) {
		foreach ( $headers as $k => $v ) {
			$this->headers[ strtolower( str_replace( '-', '_', $k ) ) ] = $v;
		}
		$this->params = $params;
	}

	public function get_header( $key ) {
		return $this->headers[ strtolower( str_replace( '-', '_', $key ) ) ] ?? null;
	}

	public function get_param( $key ) {
		return $this->params[ $key ] ?? null;
	}
}

class WP_REST_Response {
	/** @var mixed */
	public $data;
	/** @var int */
	public $status;
	/** @var array<string, string> */
	public $headers = array();

	public function __construct( $data = null, $status = 200 ) {
		$this->data   = $data;
		$this->status = $status;
	}

	public function header( $key, $value ) {
		$this->headers[ $key ] = $value;
	}

	public function get_data() {
		return $this->data;
	}
}
