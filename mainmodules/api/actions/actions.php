<?php

class apiActions extends MainActions
{
	public function executeDefaultAction()
	{
		return $this->jsonResponse(
			self::HTTP_404_NOTFOUND,
			array('error'=>'404 Not Found'));
	}

	public function jsonResponse($status,$contents)
	{
		$header = array(
			$status,
			'Content-type: application/json',
			);
		return array($header,json_encode($contents));
	}
}