<?php

class appsActions extends MainActions
{
	public function executeIndex()
	{
		//仮
		$url = mfwRequest::makeUrl('/apps/new');
		return array(array(),"<a href=\"$url\">new</a>");
	}

	public function executeNew()
	{
		$params = array(
			);
		return $this->build($params);
	}

}