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

	protected function makePackageArray(Package $pkg)
	{
		$tags = array();
		foreach($pkg->getTags() as $t){
			$tags[] = $t->getName();
		}

		return array(
			'package_url' => mfwRequest::makeUrl("/package?id={$pkg->getId()}"),
			'application_url' => mfwRequest::makeUrl("/app?id={$pkg->getAppId()}"),
			'id' => $pkg->getId(),
			'platform' => $pkg->getPlatform(),
			'title' => $pkg->getTitle(),
			'description' => $pkg->getDescription(),
			'ios_identifier' => $pkg->getIOSIdentifier(),
			'original_file_name' => $pkg->getOriginalFileName(),
			'file_size' => $pkg->getFileSize(),
			'created' => $pkg->getCreated(),
			'tags' => $tags,
			'install_count' => $pkg->getInstallCount(),
			);
	}
}

