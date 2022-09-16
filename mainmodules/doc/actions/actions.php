<?php
require_once(APP_ROOT.'/model/APKFile.php');

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

	public function executeKeystore()
	{
		$param = array(
			'info' => implode("\n", APKFile::getKeystoreInfo()),
			);
		return $this->build($param);
	}
}
