<?php
use Endroid\QrCode\QrCode;

class qrActions extends MainActions
{
    const DEFAULT_QR_SIZE = 150;
    const DEFAULT_QR_PADDING_SIZE = 10;
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
            ->setSize(self::getRenderSize($size, self::DEFAULT_QR_PADDING_SIZE))
            ->setPadding(self::DEFAULT_QR_PADDING_SIZE)
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

    private static function getRenderSize($size, $padding)
    {
        $size = $size - 2 * $padding;
        if ($size <= 0) {
            $size = self::DEFAULT_QR_SIZE;
        }
        return $size;
    }

}
