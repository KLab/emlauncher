<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Package.php';

class packageActions extends apiActions
{
	protected $pkg = null;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}

		if(!mfwRequest::has('id')){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>"id required"));
		}

		$id = (int)mfwRequest::param('id');
		$this->pkg = PackageDb::retrieveByPK($id);

		if(!$this->pkg){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>"The package (id={$id}) not found."));
		}
		if($this->app->getId()!=$this->pkg->getAppId()){
			return $this->jsonResponse(
				self::HTTP_403_FORBIDDEN,
				array('error'=>'Forbidden'));
		}

		return null;
	}

	public function executePackage_edit()
	{
		if(($has_title=mfwRequest::has('title'))){
			$title = mfwRequest::param('title');
		}
		if(($has_description=mfwRequest::has('description'))){
			$description = mfwRequest::param('description');
		}
		if(($has_protect=mfwRequest::has('protect'))){
			$protect = self::parseBool(mfwRequest::param('protect'));
		}
		if(($has_tags=mfwRequest::has('tags'))){
			$tag_names = explode(',',mfwRequest::param('tags'));
		}

		if($has_title && !$title){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>'title must not empty'));
		}

		if(!($has_title||$has_description||$has_protect||$has_tags)){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>'no parameter'));
		}

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);
			$pkg = PackageDb::retrieveByPKForUpdate($this->pkg->getId());

			if(!$has_title){
				$title = $pkg->getTitle();
			}
			if(!$has_description){
				$description = $pkg->getDescription();
			}
			if(!$has_protect){
				$protect = $pkg->isProtected();
			}
			if(!$has_tags){
				$tags = $pkg->getTags();
			}
			else{
				$tags = $app->getOrInsertTagsByName($tag_names,$con);
			}

			$pkg->updateInfo($title,$description,$protect,$tags,$con);

			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			throw $e;
		}

		return $this->jsonResponse(
			self::HTTP_200_OK,
			self::makePackageArray($pkg));
	}

	public function executePackage_attach()
	{
		$file = mfwRequest::param('file');
		if(!$file){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>'file required'));
		}

		$type = AttachedFile::getTypeFromExt(pathinfo($file['name'],PATHINFO_EXTENSION));

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);

			$attached = AttachedFileDb::insertNewAttachedFile(
				$this->pkg,$file['name'],$file['size'],$type,$con);

			$attached->uploadFile($file['tmp_name'],$file['type']);

			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			throw $e;
		}

		return $this->jsonResponse(
			self::HTTP_200_OK,
			self::makeAttachedFileArray($attached));
	}

	public function executePackage_detach()
	{
		if(!mfwRequest::has('attached_file_id')){
			return $this->jsonResponse(
				self::HTTP_400_BADREQUEST,
				array('error'=>'attached_file_id required'));
		}

		$id = (int)mfwRequest::param('attached_file_id');
		$attached = AttachedFileDb::retrieveByPK($id);

		if(!$attached || $attached->getPackageId()!=$this->pkg->getId()){
			return $this->jsonResponse(
				self::HTTP_404_NOTFOUND,
				array(
					'error'=>
					"the attached file (id=$id) not found in the package (id={$this->pkg->getId()})"));
		}

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::retrieveByPKForUpdate($this->app->getId(),$con);

			$attached->delete($con);
			$attached->deleteFile();

			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			throw $e;
		}

		return $this->jsonResponse(
			self::HTTP_200_OK,
			self::makeAttachedFileArray($attached));
	}
}
