<?php
require_once __DIR__.'/actions.php';
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';

class package_listAction extends apiActions
{
	public function executePackage_list()
	{
		try{
			$limit = (int)mfwRequest::param('limit', 20);
			$offset = (int)mfwRequest::param('offset', 0);
			$platform = mfwRequest::param('platform');
			$tags = mfwRequest::param('tags') ? explode(',', mfwRequest::param('tags')) : array();

			if (100 < $limit) {
				$limit = 100;
			}

			$app = $this->app;

			$tags = array_unique(array_filter($tags));
			$tag_set = $app->getTagsByName($tags);
			$tag_ids = $tag_set->getColumnArray('id');

			// 存在しないタグが指定された場合空を返す.
			if (count($tags) != count($tag_ids)) {
				return $this->jsonResponse(self::HTTP_200_OK,array());
			}

			apache_log('app_id',$app->getId());
			apache_log('limit',$offset);
			apache_log('offset',$offset);
			apache_log('tag_ids',$tag_ids);
			apache_log('tags',$tags);
			apache_log('platform',$platform);

			$pkgs = PackageDb::selectByAppIdPfTagsWithLimit($app->getId(), $platform, $tag_ids, $offset, $limit);

			$ret = array();
			foreach($pkgs as $pkg){
				$ret[] = self::makePackageArray($pkg);
			}
		}
		catch(Exception $e){
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			return $this->jsonResponse(
				self::HTTP_500_INTERNALSERVERERROR,
				array('error'=>$e->getMessage(),'exception'=>get_class($e)));
		}

		return $this->jsonResponse(self::HTTP_200_OK,$ret);
	}

}

