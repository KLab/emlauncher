<?php

class mypageActions extends MainActions
{

	public function executeIndex()
	{
		$params = array(
			);
		return $this->build($params);
	}
}