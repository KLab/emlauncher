<?php

class docActions extends MainActions
{

	public function executeDefaultAction()
	{
		$template = APP_ROOT.self::TEMPLATEDIR."/doc/{$this->getAction()}.php";
		if(!file_exists($template)){
			return array(array(self::HTTP_404_NOTFOUND),'404 Not Found');
		}
		return $this->build();
	}

}
