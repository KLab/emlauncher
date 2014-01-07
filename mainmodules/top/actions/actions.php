<?php
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Paging.php';

class topActions extends MainActions
{
	const LINE_IN_PAGE = 20;

	public function executeIndex()
	{
		$app_count = ApplicationDb::selectCount();
		$paging = $this->createPaging($app_count, self::LINE_IN_PAGE);

		$offset = ($paging->getCurrentPage() - 1) * self::LINE_IN_PAGE;
		$apps = ApplicationDb::selectByUpdateOrderWithLimit($offset, self::LINE_IN_PAGE);

		$params = array(
			'applications' => $apps,
			'paging' => $paging,
		);
		return $this->build($params);
	}

	protected function createPaging($item_count,$items_per_page)
	{
		$current_page = mfwRequest::param('page', 1);
		$max_page = floor(($item_count-1) / $items_per_page) + 1;
		return new Paging($current_page,$max_page);
	}

}
