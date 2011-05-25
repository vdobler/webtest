package tag

import (
	"testing"
	"fmt"
	"strings"
)


var testStructureHtml = `<html>
<body>
	<h1> A </h1>
	<p> B
		<span> C </span>
		D
	</p>
	<h2> E </h2>
	<div>
		<p> F </p>
		<p> G </p>
	</div>
</body>
</html>
`

var testXhtml = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="de" xml:lang="de">
	<head>
		<title>Some XHTML</title>
		<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
	</head>
	<body>
		<h1>X-HTML Test</h1>
		<p>The Body</p>
	</bod>
</html>
`

var testBrokenHtml1 = `<!DOCTYPE html>
<html>
<body>
	<div id="div1">
		<span id="SP1">Some aaaa text</bug>
	</wrong>
	<p>Completely Skipped</p>
</body>
</html>`

var testBrokenHtml2 = `<!DOCTYPE html>
<html>
<body>
	<div id="div1"> <!-- MyComment -->
		<span id="SP1>Some aaaa text</span>
	</div>
	<p>Some Text</p>
</body>
</html>`

var testEntitiesHtml = `<html><body>
<p>a &lt; b &gt; c. A&amp;B. x=&quot;Hallo&quot;. Copy &copy;. Umlaute: äöü = &auml;&ouml;&uuml;.</p>
</body></html>`

func testStructure(doc *Node, expected []string, t *testing.T) {
	lines := strings.Split(doc.HtmlRep(0), "\n", -1)
	for i, etag := range expected {
		a, b := "<"+etag+" ", "<"+etag+">"
		got := strings.Trim(lines[i], " \t")
		if !(strings.HasPrefix(got, a) || strings.HasPrefix(got, b)) {
			t.Errorf("Expected %s on line %d but got %s.", etag, i, got)
		}
	}
}

func testHtmlParsing(html string, expected []string, t *testing.T) {
	doc, err := ParseHtml(html)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
		t.FailNow()
	}
	testStructure(doc, expected, t)
}

//  Testcases below

func TestMostSimpleHtml(t *testing.T) {
	testHtmlParsing("<html><body>Hello</body></html>", []string{"html", "body"}, t)
}

func TestSimpleHtmlParsing(t *testing.T) {
	testHtmlParsing(testStructureHtml, []string{"html", "body", "h1", "p", "span", "h2", "div", "p", "p"}, t)
}

func TestXHtmlParsing(t *testing.T) {
	testHtmlParsing(testXhtml, []string{"html", "head", "title", "meta", "body", "h1", "p"}, t)
}

func TestHtmlEntitiesParsing(t *testing.T) {
	doc, err := ParseHtml(testEntitiesHtml)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
		t.FailNow()
	}
	lines := strings.Split(doc.HtmlRep(0), "\n", -1)
	for i, exp := range []string{"<html>", "<body>", "<p> a < b > c. A&B. x=\"Hallo\". Copy ©. Umlaute: äöü = äöü."} {
		got := strings.Trim(lines[i], " \t")
		if !strings.HasPrefix(got, exp) {
			t.Errorf("Expected %s on line %d but got %s.", exp, i, got)
		}
	}
}


func TestBrokenClosingTagParsing(t *testing.T) {
	testHtmlParsing(testBrokenHtml1, []string{"html", "body", "div", "span"}, t)
}

func TestBrokenQuoteParsing(t *testing.T) {
	LogLevel = 2
	doc, err := ParseHtml(testBrokenHtml2)
	if err == nil {
		t.Error("No error detected on broken html 2 ")
		fmt.Printf("Resulting html structure:\n%s\n", doc.HtmlRep(0))
	}
}

func BenchmarkParsing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseHtml(testSimpleHtml)
	}
}


var almostOkay = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="de" xml:lang="de">

	<head>
	<title>Kontakt (Die Direktion) Justiz-, Gemeinde- und Kirchendirektion - Kanton Bern</title>
	
<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
<meta name="copyright" content="2011 Justiz-, Gemeinde- und Kirchendirektion" />




<script src="/js/urchin.js" type="text/javascript"></script> 

<script type="text/javascript"> _userv=0; urchinTracker(); </script><link href="/etc/designs/std/css/fouc.css" rel="stylesheet" type="text/css"/>
	<style type="text/css" media="all">
		/* <![CDATA[ */
			@import url(/etc/designs/std/css/screen.css);	/* Allgemeine Definitionen */
			@import url(/etc/designs/std/css/subsite.css); 	/* Stildefinition fï¿½r Subsites */
			@import url(/sub/forms/css/calendar.css);

			@import url(/etc/designs/std/css/print.css);	    /* Fï¿½r die Printausgabe */
		/* ]]> */
	</style>
<script type="text/javascript" src="/js/jquery-1.3.2.min.js"></script>
<script type="text/javascript" src="/js/jquery-ui-1.7.2.custom.min.js"></script>
<script type="text/javascript" src="/js/i18n/ui.datepicker-de.js"></script>
<script type="text/javascript" src="/js/i18n/ui.datepicker-fr.js"></script>
<script type="text/javascript" src="/js/i18n/ui.datepicker-it.js"></script>
<script type="text/javascript" src="/js/sc_script.js"></script>

<script type="text/javascript" src="/js/std_scripts.js"></script>
<script type="text/javascript" src="/js/forms_scripts.js"></script>
<script type="text/javascript" src="/js/stema.js"></script>
<script type="text/javascript" src="/js/jquery.js"></script>
<script type="text/javascript" src="/js/ui.core.js"></script>
<script type="text/javascript" src="/js/ui.datepicker.js"></script>
<script type="text/javascript" src="/js/ui.datepicker-de.js"></script>
<script type="text/javascript">
	$(function() {
		$('.datepicker').datepicker({
			changeMonth: true,
			changeYear: true,
			showOn: 'button', 
			buttonImage: '/media/icon_agenda.gif', 
			buttonImageOnly: true
		});
	});
</script>
</head>

	<body class="">
	<a name="top"></a>
	<div id="wrapper">
		<div class="floatingComponent header">

<div id="identity">
	<img src="/etc/designs/std/media/logo.gif" alt="Kanton Bern"/>
	<h1 class="noOverwrite"><a href="http://www.be.ch/web/de">Kanton Bern <span class="hidden">Zur </span><span class="small">Startseite</span></a></h1>

			<p class="printfunctions">
				<script type="text/javascript" language="javascript">/* <![CDATA[ */ 
					document.write ("<a href='#' onclick='javascript:window.print()'>Drucken<\/a> ");
					document.write ("<a href='#' onclick='window.close()'>Schliessen<\/a>");
				/* ]]> */</script>
			</p>
	<h2><a href="/jgk/de/index.html">Justiz-, Gemeinde- und Kirchendirektion</a> <a href="/jgk/de/index.html" class="small"><span class="hidden">Zur </span>Startseite</a></h2>
</div>
<hr />

<!-- **************************************************************************
	 Sprunglink-Navigation
************************************************************************** //-->

<h1 class="hidden">Navigation</h1>
<div class="hidden">
	<h2>Sprunglinks</h2>	
	<ul>
		<li><a accesskey="0" href="/jgk/de/index.html">Direkt zur Startseite</a></li>
		<li><a accesskey="1" href="#anchor-nav-global">Direkt zur Themen- und Hauptnavigation</a></li>
		<li><a accesskey="2" href="#anchor-content">Direkt zum Inhalt</a></li>
		<li><a accesskey="5" href="#anchor-search">Direkt zur Suche</a></li>

		<li><a accesskey="6" href="#anchor-context">Zu den weiteren Informationen</a></li>
		<li><a accesskey="7" href="#anchor-nav-meta">Zur Hilfsnavigation (Kontakt, Sitemap, A bis Z)</a></li>
		<li><a accesskey="8" href="#anchor-nav-lang">Andere Sprache auswählen</a></li>			
	</ul>	
</div>

<hr />

<!-- **************************************************************************
	 Meta- & Sprach-Navigation
************************************************************************** //-->
<div class="header-meta">

	<!-- Sprach-Navigation -->
	<a name="anchor-nav-lang"></a>

	<h2 class="hidden">Andere Sprachen</h2>

	
		<div id="header-nav-lang">
		<ul>
		
				
					<li><a href="/jgk/fr/index/direktion/organisation/agr/kontakt.ssl.html">Français</a></li>
				
		
		</ul>

		</div><!-- /Sprach-Navigation -->
	
	
	<!-- Meta-Navigation -->

	<a name="anchor-nav-meta"></a>
	<h2 class="hidden">Hilfsnavigation (Kontakt, Sitemap, A bis Z)</h2>
	<div id="header-nav-meta">
		<ul>
			
				<li><a href="/jgk/de/index/direktion/ueber-die-direktion/kontakt.html">Kontakt</a></li>

			
			
				<li><a href="/jgk/de/tools/sitemap.direktion.html">Sitemap</a></li>
			
			
				<li><a href="/jgk/de/tools/a-z.html">Stichwörter von A-Z</a></li>
			
		</ul>
	</div><!-- /Meta-Navigation -->

</div><!-- / Meta- & Sprach-Navigation -->




<!-- **************************************************************************
	 Suchfeld
************************************************************************** //-->

<a name="anchor-search"></a>
<h2 class="hidden">Suche</h2>
<div id="header-search">

	<form action="/jgk/de/tools/suche.html" name="search" method="get" accept-charset="UTF-8" enctype="text/plain">
		<label for="searchform-keyword">Suche nach Stichwörtern</label>
		<input id="searchform-keyword" name="query" type="text" class="text" value="" />
		<input type="submit" class="submit" name="search-submit" value="Suchen" />
		<input type="hidden" name="reiter" value="direktion" />
		<input type="hidden" name="_charset_" value="UTF-8" />

	</form>
	
</div>
<!-- /Suchfeld -->

</div>
<!-- 
	**************************************************************************
		 Themennavigation
	**************************************************************************
//-->
<a name="anchor-nav-global"></a>		
<h2 class="hidden">Themennavigation</h2>

<div class="globalNavigation"><div id="global-nav">
	<ul>
		<li>

			<a href="/jgk/de/index/direktion.html" class="active">
				<span>Die Direktion</span>						
			</a>
			<ul>
			<li>
					<a href="/jgk/de/index/direktion/direktor.html">
						<span>Der Direktor</span>
					</a>

				</li>
				<li>
					<a href="/jgk/de/index/direktion/ueber-die-direktion.html">
						<span>&#220;ber die Direktion</span>
					</a>
				</li>
				<li>
					<span class="hidden">Sie befinden sich hier:</span>

					<a href="/jgk/de/index/direktion/organisation.html" class="active">
						<span>Organisation</span>
					</a>
				</li>
				</ul>
			</li>
		<li>
			<a href="/jgk/de/index/gemeinden/gemeinden.html">

				<span>Gemeinden</span>							
			</a>
		</li>
		<li>
			<a href="/jgk/de/index/raumplanung/raumplanung.html">
				<span>Raumplanung</span>							
			</a>
		</li>
		<li>

			<a href="/jgk/de/index/baubewilligungen/baubewilligungen.html">
				<span>Baubewilligungen</span>							
			</a>
		</li>
		<li>
			<a href="/jgk/de/index/praemienverbilligung/praemienverbilligung.html">
				<span>Pr&#228;mienverbilligung</span>							
			</a>

		</li>
		<li>
			<a href="/jgk/de/index/kinder_jugendliche/kinder_jugendliche.html">
				<span>Kinder &amp; Jugendliche</span>							
			</a>
		</li>
		<li>
			<a href="/jgk/de/index/kirchen/kirchen.html">

				<span>Kirchen</span>							
			</a>
		</li>
		<li>
			<a href="/jgk/de/index/aufsicht.html">
				<span>Aufsicht</span>							
			</a>
		</li>
		</ul>

	</div>
</div>
<!-- **************************************************************************
	 Hauptbereich
************************************************************************** //-->
<a name="anchor-nav-main"></a>
<div id="content">
	<!-- **************************************************************************
		Linke Spalte
************************************************************************** //-->
<div id="content-col-nav">


	<!-- **************************************************************************
		 Hauptnavigations-Block
	************************************************************************** //-->
  
  <h2 class="hidden">Hauptnavigation</h2>
  
  <div class="hauptNavigation"><ul>

		       <li>
     
     <a href="/jgk/de/index/direktion/organisation/organigramm.html">Organigramm</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/gs.html">Generalsekretariat</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/aba.html">Amt für Betriebswirtschaft und Aufsicht</a>

      </li>
    <li>
     
     <a class="active" href="/jgk/de/index/direktion/organisation/agr.html">Amt für Gemeinden und Raumordnung</a>
         <ul>
		       <li>
     
     <a href="/jgk/de/index/direktion/organisation/agr/aktuell.html">Aktuell</a>
      </li>
    <li>

     
     <a href="/jgk/de/index/direktion/organisation/agr/ueber_uns.html">Über uns</a>
      </li>
    <li>
     
     <span class="hidden">Sie befinden sich hier:</span>
     <a class="active current" href="/jgk/de/index/direktion/organisation/agr/kontakt.html">Kontakt</a>
         </li>
    <li>

     
     <a href="/jgk/de/index/direktion/organisation/agr/organigramm.html">Organigramm</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/agr/rechtliche_grundlagen.html">Rechtliche Grundlagen</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/agr/formulare_bewilligungen.html">Formulare / Bewilligungen</a>

      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/agr/downloads_publikationen.html">Downloads &amp; Publikationen</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/agr/offene_stellen.html">Offene Stellen</a>
      </li>

	    </ul>
	  </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/asvs.html">Amt für Sozialversicherung und Stiftungsaufsicht</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/bka.html">Beauftragter für kirchliche Angelegenheiten</a>
      </li>

    <li>
     
     <a href="/jgk/de/index/direktion/organisation/kja.html">Kantonales Jugendamt</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/ra.html">Rechtsamt</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/baka.html">Betreibungs- und Konkursämter</a>

      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/gba.html">Grundbuchämter</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/hra.html">Handelsregisteramt des Kantons Bern</a>
      </li>
    <li>

     
     <a href="/jgk/de/index/direktion/organisation/rsta.html">Regierungsstatthalterämter</a>
      </li>
    <li>
     
     <a href="/jgk/de/index/direktion/organisation/dsa.html">Datenschutzaufsichtsstelle</a>
      </li>
	    </ul>
	  </div>
</div><!-- /Linke Spalte -->

<hr />
	<!-- **************************************************************************
		Mittlere Spalte
************************************************************************** //-->

<a name="anchor-content"></a>
<div id="content-col-main">

	<!-- Banner -->
	<div class="parbase"></div>
<!-- **************************************************************************
		 Breadcrumb-Trail
	************************************************************************** //-->
	<div class="breadcrumb"><div id="breadcrumb">
	<h2 class="hidden">Sie befinden sich derzeit auf folgender Seite:</h2>

		<a href="/jgk/de/index.html">Startseite</a>&nbsp;&gt;&nbsp;
			<a href="/jgk/de/index/direktion.html">Die Direktion</a>&nbsp;&gt;&nbsp;
			<a href="/jgk/de/index/direktion/organisation.html">Organisation</a>&nbsp;&gt;&nbsp;
			<a href="/jgk/de/index/direktion/organisation/agr.html">Amt für Gemeinden und Raumordnung</a>&nbsp;&gt;&nbsp;
			<span>Kontakt</span>
	</div>

</div>
<hr />

	<!-- Drucken-Link -->
	<!-- **************************************************************************
		 Drucken-Link
	************************************************************************** //-->
	
	<div id="content-print">
		<a href="/jgk/de/index/direktion/organisation/agr/kontakt.ssl.print.html" target="_blank" onclick="window.open('/jgk/de/index/direktion/organisation/agr/kontakt.ssl.print.html','popup_print','width=600,height=800,scrollbars=yes,resizable=yes');return false" rel="nofollow" class="intern">Seite drucken</a>
	</div><!-- /Drucken-Link --><!-- **************************************************************************
		 Beginn Seiteninhalt
	************************************************************************** //-->

	<div class="content">

		



	<div>
		<h1>Kontakt</h1>
		
	</div>
<div class="designBild parbase">
<!-- Don't show the image in preview mode if it is empty. -->

</div>
<div class="contentNavigation"></div>
<div class="ankerNavigation">




</div>
<div class="middlePar parsys parSys"><div id="parsys">
		<a name="middlePar_textbild_4948" style="visibility:hidden"></a>
	      <div class="textBild floatingComponent section"><h2><strong>Adresse</strong></h2>
<p>Amt für Gemeinden und Raumordnung<br />
Nydeggasse 11/13<br />
3011 Bern</p>
<p>Telefon 031 633 77 30<br />
Telefax 031 633 77 31</p>

<p><a href="/schutz/info.agr/jgk.be/ch" target="techframe" rel="nofollow"><span class="ieicon">&bull;</span>E-Mail</a></p>
<p>Französischsprachige Verwaltungsstelle des AGR<br />
Hauptstrasse 2<br />
Postfach<br />
2560 Nidau</p>
<p>Telefon 032 329 88 00<br />
Telefax 032 329 88 30<br />
</p>
<p><a href="/schutz/oacot/jgk.be/ch" target="techframe" rel="nofollow"><span class="ieicon">&bull;</span>Email</a><br />

</p>
<h2>Öffnungszeiten<br />
</h2>
<p>Montag - Donnerstag 08.00 - 12.00 Uhr, 13.30 - 17.00 Uhr<br />
Freitag bis 16.00 Uhr<br />
</p>
<p><a href="/jgk/de/index/direktion/organisation/agr/kontakt.assetref/content/dam/documents/JGK/AGR/de/organisation/agr_portrait_stao_bern.jpg" target="_blank" class="image"><span class="ieicon">&bull;</span>Situationsplan Bern</a> <span class="info">(JPG, 22&nbsp;KB)</span><br />
<a href="/jgk/de/index/direktion/organisation/agr/kontakt.assetref/content/dam/documents/JGK/AGR/de/organisation/AGR%20Nidau_d.JPG" target="_blank" class="image"><span class="ieicon">&bull;</span>Situationsplan Nidau</a> <span class="info">(JPG, 387&nbsp;KB)</span><br />

<br />
</p>
<h2>Öffentliche Verkehrsmittel<br />
</h2>
<p>Bernmobil ab Bahnhof Bus-Linie 12 bis Haltestelle Nydegg<br />
Bernmobil ab Bahnhof Tram-Linien 6, 7, 8, 9 oder Bus-Linie 10 bis Haltestelle Zytglogge, dann weiter zu Fuss oder ab Zytglogge Bus-Linie 12 bis Haltestelle Nydegg</p>
<h2>Mit dem Auto<br />
</h2>
<p>Parkplätze beim Bärengraben, Nydeggasse oder Nydeggstalden sowie in den Parkhäusern gegen Gebühr / blaue Zone&nbsp;vorhanden. <br />
</p>
<div class="clear"></div></div>

<a name="middlePar_textbild" style="visibility:hidden"></a>
	      <div class="textBild floatingComponent section"><h2>Kontaktformular für das Amt für Gemeinden und Raumordnung (AGR)</h2>
<div class="clear"></div></div>
<a name="middlePar_kontaktformular" style="visibility:hidden"></a>
	      <div class="floatingComponent section kontaktformular">

	<form enctype="multipart/form-data" method="post" accept-charset="UTF-8" action="/jgk/de/index/direktion/organisation/agr/kontakt.kontaktform.html">
		<input type="hidden" name="_charset_" value="UTF-8" />
		<input type="hidden" name="resource" value="/content/jgk/de/index/direktion/organisation/agr/kontakt/jcr:content/middlePar/kontaktformular" />
		
		
		<p>* Pflichtfelder</p>

        
            
            
            
        		<div class="row">
        			<label for="form-theme">Betreff *</label>
        			
        				
        				
        					
        				
        						
        			<select class="full" name="theme" id="form-theme">
        				
        					
        						<option value="" selected="selected">Bitte wählen...</option>
        					
        					
        				
        				
        					
        						
        						
        							<option value="94">Allgemeine Anfrage</option>
        						
        					
        				
        					
        						
        						
        							<option value="96">Gemeinden</option>
        						
        					
        				
        					
        						
        						
        							<option value="98">Kantonsplanung und kantonaler Richtplan</option>

        						
        					
        				
        					
        						
        						
        							<option value="100">Orts- und Regionalplanung</option>
        						
        					
        				
        					
        						
        						
        							<option value="102">Baubewilligungen</option>
        						
        					
        				
        					
        						
        						
        							<option value="104">Französischsprachige Verwaltungsstelle</option>
        						
        					
        				
        			</select>
        		</div>
            
        
		
			
			
				
			
		
		<div class="row">
			<label for="form-comment">Kommentar *</label>

			<textarea cols="30" rows="5" name="comment" id="form-comment" class="text full"></textarea>
		</div>
		
			
			
				
			
		
		<div class="row no-label">
			
				
			
			<input type="checkbox" value="erwünscht" id="form-contact-me" name="contact-me" class="checkbox" checked />
			<label class="checkbox" for="form-contact-me">Ich möchte, dass Sie mich kontaktieren.</label>
		</div>
		<div class="row no-label">
			
			
			
			<input type="radio" value="männlich" id="form-gender-m" name="gender" class="checkbox"  />

			<label class="checkbox" for="form-gender-w">Herr</label>
			<input type="radio" value="weiblich" id="form-gender-w" name="gender" class="checkbox"  /> 
			<label class="checkbox" for="form-gender-w">Frau</label>
		</div>
		<div class="row">
			<label for="form-prename">Vorname</label> 
			<input type="text" name="prename" id="form-prename" class="text large" value="" />
		</div>

		
		
			
			
				
			
		
		<div class="row">
			
			<label for="form-lastname">Nachname * <span class="hidden">
				
					
					
						Pflichtfeld
					
				
				</span></label> 
			<input type="text" name="lastname" id="form-lastname" class="text large" value="" />
		</div>
		
		<div class="row">
			<label for="form-firm">Firma/Organisation</label>
			<input type="text" name="firm" id="form-firm" class="text large" value="" />

		</div>
		<div class="row">
			<label for="form-street">Strasse und Hausnummer</label> 
			<input type="text" name="street" id="form-street" class="text large" value="" />
		</div>
		<div class="row">
			<label for="form-zip">PLZ</label> 
			<input type="text" name="zip" id="form-zip" class="text small" value="" />
		</div>

		<div class="row">
			<label for="form-city">Ort</label> 
			<input type="text" name="city" id="form-city" class="text large" value="" />
		</div>

		
			
			
				
			
		
		<div class="row">
			
			<label for="form-mail">E-Mail * <span class="hidden">
				
					
					
						Pflichtfeld
					
				
				</span></label> 
			<input type="text" name="Email" id="form-mail" class="text large" value="" />

		</div>
		
		
		
			
			
				
			
		
		<div class="row">
			
			<label for="form-phone">Telefon * <span class="hidden">
				
					
					
						Pflichtfeld
					
				
				</span></label> 
			<input type="text" name="phone" id="form-phone" class="text large" value="" />
		</div>
		
		<div class="row">
			<input type="submit" value="Senden" name="submit" class="submit" />

		</div>
	
		<!-- Verstecktes Formularfeld zum Spamschutz -->
		<div class="row hidden">
			<label for="form-feld">
				Leeres Eingabefeld (Dieses Feld bitte nicht ausfüllen, es dient lediglich dazu, dieses Formular vor Massenmailmissbrauch zu schüzen.)
			</label> 
			<input type="text" value="f" id="form-feld" name="feld" />
			<input type="text" value="" id="form-feld2" name="feld2" />
		</div>
	</form>

 
 
</div>
</div>
  </div>
<!-- Top-Link -->
		<div class="nachOben floatingComponent">
<a class="top" href="#top">Nach oben</a></div>
</div><!-- /Seiteninhalt -->

</div><!-- /Mittlere Spalte -->
<hr />
	<!-- **************************************************************************
	Rechte Spalte
************************************************************************** //-->

<a name="anchor-context"></a>
<h1 class="hidden">Weitere Informationen</h1>
<div id="content-col-context" style="background-image: url(/jgk/de/index/direktion/organisation/_jcr_content/keyVisual/image.210!.75!.png)">
		<div class="keyVisual parbase ktbeImage">

	
	

</div>
<div class="kampagneTeaser"></div>
<div class="floatingComponent seitenspezifischerKontakt">




	
		
		
		
				
					
						<div class="section">

					
					
				
					<div class="kontaktComponent contact"><div class="box contact">                 
			<div class="title">
					<h2>Kontakt</h2>
				</div>
		
				<div class="body">
			
					<h3>Justiz-, Gemeinde- und Kirchendirektion </h3>
					<p>
						Amt für Gemeinden und Raumordnung<br />

						Nydeggasse 11/13<br />
						3011 Bern</p>
						<p>
						Tel. 031 633 77 30<br />
						Fax 031 633 77 31<br />
						<a href="/schutz/info.agr/jgk.be/ch" class="rewrite-noicon" target="techframe" rel="nofollow">Kontakt per E-Mail</a><br />

						<a href="/jgk/de/index/direktion/organisation/agr/kontakt.html">Kontaktformular</a>
						</p>
					
					<!--  mittlere box -->
					<div class="lined">
							<p><a href="http://map.search.ch/bern/nydeggasse-11" target="_blank" title="Link öffnet in einem neuen Fenster." class="extern">Situationsplan</a></p>
</div>
					<!-- ende mittlere box -->
		
					</div>

			</div>
		
		<div style="clear:both">&nbsp;</div>	




</div>

				</div>				
		
	
	
</div>
</div>
  <br class="clear" /><!-- /Rechte Spalte -->
<hr />
	<div class="clear"></div>
</div><!-- /Hauptbereich -->
</div><!-- /wrapper -->

	<!-- **************************************************************************
	Footer
************************************************************************** //-->
<h1 class="hidden">Informationen über diesen Webauftritt</h1>
<div id="footer">
	<div class="footer floatingComponent">
	<p>
		&copy;
		<a href="/jgk/de/index.html">2011 Justiz-, Gemeinde- und Kirchendirektion</a>
	</p>
	
	  <p class="links"><a href="/jgk/de/tools/impressum.html">Impressum</a> <a href="/jgk/de/tools/rechtliches.html">Rechtliches</a></p>

	

	<p class="printfuntions">
		<script type="text/javascript" language="javascript">/* <![CDATA[ */ 
			document.write ("<a href='#' onclick='javascript:window.print()'>Drucken<\/a> ");
			document.write ("<a href='#' onclick='window.close()' class='rewrite-noicon'>Schliessen<\/a>");
		/* ]]> */</script>
	</p>
	
	<iframe style="display:none" name="techframe"></iframe></div>
</div>
<!-- **************************************************************************
		URL
	************************************************************************** //-->
	<p id="url">http://www.jgk.be.ch/de/index/direktion/organisation/agr/kontakt</p>
</body>
</html>

`

func TestAlmostOkay(t *testing.T) {
	_, err := ParseHtml(almostOkay)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
		t.FailNow()
	}
	// fmt.Print(doc.HtmlRep(2))
}
