<!DOCTYPE html>
<html lang="ja">
  <head>
    <meta charset="utf-8">
    <title><?=(isset($page_title))?$title.' | ':''?>EM-Launcher</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
<?php if(mfwServerEnv::getEnv()==='local'): ?>
    <link href="/bootstrap/bootswatch/spacelab/bootstrap.min.css" rel="stylesheet" media="screen">
    <script src="/jquery/jquery.js"></script>
    <script src="/bootstrap/3.0.0/js/bootstrap.min.js"></script>
    <link href="/font-awesome/4.0.1/css/font-awesome.min.css" rel="stylesheet">
<?php else: ?>
    <link href="//netdna.bootstrapcdn.com/bootswatch/3.0.0/spacelab/bootstrap.min.css" rel="stylesheet" media="screen">
    <script src="//code.jquery.com/jquery.js"></script>
    <script src="//netdna.bootstrapcdn.com/bootstrap/3.0.0/js/bootstrap.min.js"></script>
    <link href="//netdna.bootstrapcdn.com/font-awesome/4.0.1/css/font-awesome.min.css" rel="stylesheet">
<?php endif ?>
    <link rel="stylesheet" href="<?=url('/css/customize.css')?>" type="text/css">
  </head>
  <body>

    <div class="navbar navbar-inverse" role="navigation">
      <div class="navbar-header">
        <button type="button" class="navbar-toggle" data-toggle="collapse" data-target=".navbar-ex1-collapse">
          <span class="sr-only">Toggle navigation</span>
          <span class="icon-bar"></span>
          <span class="icon-bar"></span>
          <span class="icon-bar"></span>
        </button>
        <a href="<?=url('/mypage')?>" class="navbar-brand"><span>EM</span><span>Launcher</span></a>
      </div>
      <div class="collapse navbar-collapse navbar-ex1-collapse">
<?php if($login_user): ?>
        <ul class="nav navbar-nav">
          <li><a href="<?=url('/project')?>">Projects</a></li>
        </ul>
        <ul class="nav navbar-nav navbar-right">
          <li class="dropdown">
            <a href="#" class="dropdown-toggle" data-toggle="dropdown"><?=$login_user->getMail()?> <b class="caret"></b></a>
            <ul class="dropdown-menu">
              <li><a href="<?=url('/mypage')?>">Mypage</a></li>
              <li><a href="<?=url('/project/create')?>">New project</a></li>
              <li><a href="<?=url('/logout')?>">Logout</a></li>
            </ul>
          </li>
        </ul>
<?php else: ?>
        <ul class="nav navbar-nav navbar-right">
          <li><a href="<?=url('/login')?>">Login</a></li>
        </ul>
<?php endif ?>
      </div>
    </div>

    <?=$contents?>
  </body>
</html>