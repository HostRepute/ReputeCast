<?php
/**
 * What the plugin keeps.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Functions;
use ReputeCast\Options;
use ReputeCast\Tests\TestCase;

final class OptionsTest extends TestCase {

	public function test_the_key_is_kept_encrypted_and_not_autoloaded(): void {
		$autoload = null;
		Functions\when( 'update_option' )->alias(
			function ( $name, $value, $load = null ) use ( &$autoload ) {
				$this->options[ $name ] = $value;
				if ( Options::KEY === $name ) {
					$autoload = $load;
				}
				return true;
			}
		);
		Options::set_key( self::KEY );
		$stored = $this->options[ Options::KEY ];
		$this->assertStringStartsWith( 'v1:', $stored );
		$this->assertStringNotContainsString( 'rc_live_', $stored );
		$this->assertStringNotContainsString( substr( self::KEY, 8 ), base64_decode( substr( $stored, 3 ) ) );
		$this->assertFalse( $autoload, 'the key option is not autoloaded' );
		$this->assertSame( self::KEY, Options::key() );

		// Another nonce each time.
		Options::set_key( self::KEY );
		$this->assertNotSame( $stored, $this->options[ Options::KEY ] );
	}

	public function test_a_copy_of_the_database_alone_doesnt_reveal_the_key(): void {
		Options::set_key( self::KEY );
		// Another site's salts (wp-config.php) can't open it.
		Functions\when( 'wp_salt' )->justReturn( 'another-site-salt' );
		$this->assertSame( '', Options::key() );
	}

	public function test_a_damaged_key_reads_as_none(): void {
		$this->options[ Options::KEY ] = 'v1:' . base64_encode( str_repeat( 'x', 60 ) );
		$this->assertSame( '', Options::key() );
		$this->options[ Options::KEY ] = 'v1:not base64!';
		$this->assertSame( '', Options::key() );
		$this->options[ Options::KEY ] = self::KEY;
		$this->assertSame( '', Options::key(), 'a key stored in clear by hand is not used' );
	}

	public function test_the_key_as_lists_show_it(): void {
		Options::set_key( self::KEY );
		$this->assertSame( 'rc_live_5d1e…7b20', Options::masked_key() );
		Options::forget_key();
		$this->assertSame( '', Options::masked_key() );
	}

	public function test_connected_takes_a_key_an_address_and_a_workspace(): void {
		$this->assertFalse( Options::is_connected() );
		$this->connect_site();
		$this->assertTrue( Options::is_connected() );
		Options::forget_key();
		$this->assertFalse( Options::is_connected() );
		// The rest stays, so forms keep loading.
		$this->assertSame( self::INSTANCE, Options::connection()['instance_url'] );
	}

	public function test_the_connection_has_its_defaults(): void {
		$this->options[ Options::CONNECTION ] = 'garbage';
		$c                                    = Options::connection();
		$this->assertSame( '', $c['instance_url'] );
		$this->assertSame( array(), $c['forms'] );
		Options::update( array( 'site' => 'blog.acme.io' ) );
		$this->assertSame( 'blog.acme.io', Options::connection()['site'] );
	}

	public function test_uninstalling_removes_everything(): void {
		$this->connect_site();
		Options::delete_all();
		$this->assertSame( array(), $this->options );
	}
}
