<?php

/**
 * Row object for 'application' table.
 */
class Paging {

	protected $current_page_number;
	protected $total_page_number;
	protected $line_in_page;
	protected $candidate_page;

	function __construct($current_page_number = 1, $total_item_number = 1, $line_in_page = 10) {
		$this->line_in_page = $line_in_page;
		$this->total_page_number = (int)ceil(max($total_item_number, 1) / $this->line_in_page);
		if (!is_numeric($current_page_number) || $current_page_number <= 0 || $current_page_number > $this->getTotalPageNumber()) {
			$current_page_number = 1;
		}
		$this->current_page_number = $current_page_number;
		$this->candidate_page = 5;
	}
	public function getCurrentPage() {
		return $this->current_page_number;
	}
	public function getTotalPageNumber() {
		return $this->total_page_number;
	}
	public function getPageStartOffset($page) {
		return ($page - 1) * $this->line_in_page;
	}
	public function getCandidatePageNumber() {
		return $this->candidate_page;
	}
}