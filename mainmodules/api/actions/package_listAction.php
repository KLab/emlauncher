<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';

class package_listAction extends apiActions
{
	public function executePackage_list()
	{
		try{
			$api_key = mfwRequest::param('api_key');
			$limit = (int)mfwRequest::param('limit', 20);
			$platform = mfwRequest::param('platform');
			$tags = array();

			if (100 < $limit) {
				$limit = 100;
			}

			$app = ApplicationDb::selectByApiKey($api_key);
			if(!$app){
				return $this->jsonResponse(
					self::HTTP_400_BADREQUEST,
					array('error'=>'Invalid api_key'));
			}

			$pkgs = PackageDb::selectByAppIdPfTagsWithLimit($app->getId(), $platform, $tags, 0, $limit);

			$ret = array();
			foreach($pkgs as $pkg){
				$ret[] = $this->makePackageArray($pkg);
			}
		}
		catch(Exception $e){
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage(),'exception'=>get_class($e)));
		}

		apache_log('app_id',$app->getId());

		return $this->jsonResponse(self::HTTP_200_OK,$ret);
	}

}

