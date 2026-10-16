<?php
/**
 * The contract between the plugin and ReputeCast's public API.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests;

/**
 * Checks the requests the plugin makes, and the answers its tests mock,
 * against the OpenAPI document the server generates (web/openapi.json):
 * the operation exists, and the body validates against its schema. Request
 * bodies are strict (additionalProperties: false), so a field the server
 * doesn't know fails here, not on a user's site. The same checks as
 * integrations/contract/openapi-contract.mjs, for the JavaScript apps.
 */
final class Contract {

	/**
	 * The document, decoded with objects as stdClass.
	 *
	 * @var object|null
	 */
	private static $spec = null;

	/** Path of the OpenAPI document. */
	public static function path(): string {
		return dirname( __DIR__, 3 ) . '/web/openapi.json';
	}

	/** The OpenAPI document. */
	public static function spec(): object {
		if ( null === self::$spec ) {
			self::$spec = json_decode( (string) file_get_contents( self::path() ), false, 512, JSON_THROW_ON_ERROR );
		}
		return self::$spec;
	}

	/**
	 * Every operation: operationId → { method, path, op }.
	 *
	 * @return array<string, array{method: string, path: string, op: object}>
	 */
	public static function operations(): array {
		$out = array();
		foreach ( get_object_vars( self::spec()->paths ) as $path => $item ) {
			foreach ( array( 'get', 'put', 'post', 'delete', 'patch' ) as $m ) {
				if ( isset( $item->$m->operationId ) ) {
					$out[ $item->$m->operationId ] = array(
						'method' => strtoupper( $m ),
						'path'   => $path,
						'op'     => $item->$m,
					);
				}
			}
		}
		return $out;
	}

	/**
	 * The operation a request calls, with its path parameters, or null.
	 * Literal segments beat parameters.
	 *
	 * @param string $method HTTP method.
	 * @param string $url    The URL called.
	 * @return array{operationId: string, method: string, path: string, op: object, params: array<string, string>}|null
	 */
	public static function find( string $method, string $url ): ?array {
		$path = (string) parse_url( $url, PHP_URL_PATH );
		$best = null;
		foreach ( self::operations() as $id => $o ) {
			if ( strtoupper( $method ) !== $o['method'] ) {
				continue;
			}
			$names = array();
			$re    = preg_replace_callback(
				'/\\\\\{([^}]+)\\\\\}/',
				static function ( $m ) use ( &$names ) {
					$names[] = $m[1];
					return '([^/]+)';
				},
				preg_quote( $o['path'], '#' )
			);
			if ( ! preg_match( '#^' . $re . '$#', $path, $hit ) ) {
				continue;
			}
			$params = array();
			foreach ( $names as $i => $name ) {
				$params[ $name ] = rawurldecode( $hit[ $i + 1 ] );
			}
			if ( null === $best || count( $params ) < count( $best['params'] ) ) {
				$best = array( 'operationId' => $id ) + $o + array( 'params' => $params );
			}
		}
		return $best;
	}

	/**
	 * Follows $refs.
	 *
	 * @param mixed $schema A schema.
	 * @return object|null
	 */
	private static function resolve( $schema ): ?object {
		$s = is_object( $schema ) ? $schema : null;
		for ( $i = 0; null !== $s && isset( $s->{'$ref'} ) && $i < 20; $i++ ) {
			$name = str_replace( '#/components/schemas/', '', $s->{'$ref'} );
			$s    = self::spec()->components->schemas->$name ?? null;
		}
		return $s;
	}

	/**
	 * A JSON value's type.
	 *
	 * @param mixed $v A decoded value.
	 */
	private static function type_of( $v ): string {
		if ( null === $v ) {
			return 'null';
		}
		if ( is_array( $v ) ) {
			return 'array';
		}
		if ( is_object( $v ) ) {
			return 'object';
		}
		if ( is_int( $v ) ) {
			return 'integer';
		}
		if ( is_float( $v ) ) {
			return 'number';
		}
		return is_bool( $v ) ? 'boolean' : 'string';
	}

	/**
	 * Validates a value (decoded with objects as stdClass) against a schema.
	 *
	 * @param mixed  $schema The schema.
	 * @param mixed  $value  The value.
	 * @param string $at     Where, for the messages.
	 * @return array<int, string> The problems.
	 */
	public static function validate( $schema, $value, string $at = 'body' ): array {
		$s = self::resolve( $schema );
		if ( null === $s ) {
			return array();
		}
		$got   = self::type_of( $value );
		$types = isset( $s->type ) ? (array) $s->type : array();
		if ( $types && ! in_array( $got, $types, true ) && ! ( 'integer' === $got && in_array( 'number', $types, true ) ) ) {
			return array( "$at: expected " . implode( ' or ', $types ) . ", got $got" );
		}
		$errors = array();
		if ( isset( $s->enum ) && ! in_array( $value, $s->enum, true ) ) {
			$errors[] = "$at: " . json_encode( $value ) . ' is not one of ' . json_encode( $s->enum );
		}
		if ( is_string( $value ) ) {
			$n = mb_strlen( $value, 'UTF-8' );
			if ( isset( $s->minLength ) && $n < $s->minLength ) {
				$errors[] = "$at: shorter than {$s->minLength}";
			}
			if ( isset( $s->maxLength ) && $n > $s->maxLength ) {
				$errors[] = "$at: longer than {$s->maxLength}";
			}
			if ( isset( $s->pattern ) && ! preg_match( '/' . str_replace( '/', '\/', $s->pattern ) . '/u', $value ) ) {
				$errors[] = "$at: doesn't match {$s->pattern}";
			}
			if ( 'date-time' === ( $s->format ?? '' ) && ! preg_match( '/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/', $value ) ) {
				$errors[] = "$at: not an RFC 3339 date-time";
			}
		}
		if ( is_int( $value ) || is_float( $value ) ) {
			if ( isset( $s->minimum ) && $value < $s->minimum ) {
				$errors[] = "$at: below {$s->minimum}";
			}
			if ( isset( $s->maximum ) && $value > $s->maximum ) {
				$errors[] = "$at: above {$s->maximum}";
			}
		}
		if ( is_array( $value ) ) {
			if ( isset( $s->minItems ) && count( $value ) < $s->minItems ) {
				$errors[] = "$at: fewer than {$s->minItems} items";
			}
			if ( isset( $s->maxItems ) && count( $value ) > $s->maxItems ) {
				$errors[] = "$at: more than {$s->maxItems} items";
			}
			if ( isset( $s->items ) ) {
				foreach ( $value as $i => $item ) {
					$errors = array_merge( $errors, self::validate( $s->items, $item, "{$at}[{$i}]" ) );
				}
			}
		}
		if ( is_object( $value ) ) {
			foreach ( $s->required ?? array() as $r ) {
				if ( ! property_exists( $value, $r ) ) {
					$errors[] = "$at.$r: required";
				}
			}
			$props = isset( $s->properties ) ? get_object_vars( $s->properties ) : array();
			foreach ( get_object_vars( $value ) as $k => $v ) {
				if ( array_key_exists( $k, $props ) ) {
					$errors = array_merge( $errors, self::validate( $props[ $k ], $v, "$at.$k" ) );
				} elseif ( isset( $s->additionalProperties ) && false === $s->additionalProperties ) {
					$errors[] = "$at.$k: not a property the server accepts";
				} elseif ( isset( $s->additionalProperties ) && is_object( $s->additionalProperties ) ) {
					$errors = array_merge( $errors, self::validate( $s->additionalProperties, $v, "$at.$k" ) );
				}
			}
		}
		return $errors;
	}

	/**
	 * Checks a request: the operation, its path parameters and its JSON body.
	 *
	 * @param string $method HTTP method.
	 * @param string $url    The URL called.
	 * @param string $body   The JSON body sent ('' for none).
	 * @return array{operationId: string, errors: array<int, string>}
	 */
	public static function check_request( string $method, string $url, string $body = '' ): array {
		$found = self::find( $method, $url );
		if ( null === $found ) {
			return array(
				'operationId' => '',
				'errors'      => array( "$method " . parse_url( $url, PHP_URL_PATH ) . ': no such operation' ),
			);
		}
		$errors = array();
		foreach ( $found['op']->parameters ?? array() as $p ) {
			if ( 'path' === $p->in ) {
				$v = $found['params'][ $p->name ] ?? '';
				if ( '' === $v ) {
					$errors[] = "path.{$p->name}: missing";
				} else {
					$errors = array_merge( $errors, self::validate( $p->schema ?? null, $v, "path.{$p->name}" ) );
				}
			}
		}
		$content = $found['op']->requestBody->content->{'application/json'} ?? null;
		if ( null !== $content ) {
			if ( '' === $body ) {
				if ( ! empty( $found['op']->requestBody->required ) ) {
					$errors[] = 'body: required';
				}
			} else {
				$errors = array_merge( $errors, self::validate( $content->schema, json_decode( $body, false, 512, JSON_THROW_ON_ERROR ) ) );
			}
		} elseif ( '' !== $body ) {
			$errors[] = 'body: the operation takes no body';
		}
		return array(
			'operationId' => $found['operationId'],
			'errors'      => $errors,
		);
	}

	/**
	 * Checks a mocked answer against the operation's documented answer.
	 * Problems (status 400 and up) may carry fields of their own.
	 *
	 * @param string $operation_id The operation.
	 * @param int    $status       HTTP status.
	 * @param string $body         The JSON answer.
	 * @return array<int, string> The problems.
	 */
	public static function check_response( string $operation_id, int $status, string $body ): array {
		$o = self::operations()[ $operation_id ] ?? null;
		if ( null === $o ) {
			return array( "$operation_id: no such operation" );
		}
		$responses = $o['op']->responses;
		$r         = $responses->{(string) $status} ?? ( $status >= 400 ? ( $responses->default ?? null ) : null );
		if ( null === $r ) {
			return array( "$operation_id: no $status answer documented" );
		}
		$content = $r->content->{'application/json'} ?? $r->content->{'application/problem+json'} ?? null;
		if ( null === $content ) {
			return '' === $body ? array() : array( "$operation_id: the $status answer has no body" );
		}
		$schema = $content->schema;
		if ( $status >= 400 ) {
			$schema = clone self::resolve( $schema );
			unset( $schema->additionalProperties );
		}
		return self::validate( $schema, json_decode( $body, false, 512, JSON_THROW_ON_ERROR ), 'response' );
	}
}
