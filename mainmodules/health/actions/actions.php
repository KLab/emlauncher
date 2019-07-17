<?php

class healthActions extends MainActions
{
	public function initialize()
	{
		return null;
	}

	public function executeIndex()
	{
		return $this->response(self::HTTP_200_OK,'OK');
	}
}
