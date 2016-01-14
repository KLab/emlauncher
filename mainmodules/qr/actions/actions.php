<?php
require_once (APP_ROOT.'/libs/QrCode/src/QrCode.php');
use Endroid\QrCode\QrCode;

class qrActions extends MainActions
{
    /**
     * EMLauncher用のQRコードを生成します。
     * 渡されるべき q は "/app?id=1"のようなQueryStringのみ
     * Action側でEMLauncherのHOSTを追加してQRコードを生成します
     *
     * @return array
     * @throws \Endroid\QrCode\Exceptions\ImageFunctionUnknownException
     */
    public function executeCode()
    {
        $query = mfwRequest::param('q', "");
        $url = mfwRequest::makeUrl($query);
        $size =(int)mfwRequest::param('s', 150);
        if ($size > 300) {
            $size = 300;
        }
        $qr_code = new QrCode();
        $qr_code
            ->setText($url)
            ->setSize($size)
            ->setPadding(10)
            ->setErrorCorrection('high');
        return $this->pngResponse(
            $qr_code->get()
        );
    }

    private function pngResponse($contents)
    {
        $header = array(
            'Content-type: image/png',
        );
        return array($header,$contents);
    }

}
