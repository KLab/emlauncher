<?php
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Paging.php';

class topActions extends MainActions
{
	const LINE_IN_PAGE = 1;

	public function executeIndex()
	{
		$current_page = mfwRequest::param('page', 1, 'GET');
		$paging = new Paging($current_page, ApplicationDb::selectCount(''), self::LINE_IN_PAGE);
		$apps = ApplicationDb::selectAllByUpdateOrderWithLimit($paging->getPageStartOffset($current_page), self::LINE_IN_PAGE);
		$params = array(
			'applications' => $apps,
			'paging' => $paging,
		);
		return $this->build($params);
	}
}