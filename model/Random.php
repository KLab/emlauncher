<?php

class Random {

	public static function string($length=16)
	{
		static $str = '1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
		$max = strlen($str)-1;

		$r = '';
		for($i=0;$i<$length;++$i){
			$r .= $str[mt_rand(0,$max)];
		}
		return $r;
	}

}

