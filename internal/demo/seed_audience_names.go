package demo

import "strings"

// Names for generated contacts: a mix of the regions a newsletter like
// Acme's reaches. Only ASCII is used in addresses.
var firstNames = strings.Fields(`
	Aaron Abigail Adaeze Adam Adrian Ahmed Aiko Alejandro Alex Alice Alicia Amara Amelia Amir Ana Andre Andrea Anna Anton Arjun
	Ava Ayumi Beatriz Ben Benjamin Bianca Bilal Brian Bruno Camila Carla Carlos Caroline Chen Chiara Chloe Chris Claire Clara Daniel
	Daria David Diana Dmitri Elena Elif Elijah Elisa Emeka Emily Emma Erik Ethan Eva Fatima Felix Fernando Fiona Francesca Gabriel Grace
	Hana Hannah Harper Hiroshi Hugo Ibrahim Igor Ines Isaac Isabel Ivan Jack Jacob Jamal James Jana Javier Jin Joao Jonas Jorge Jose
	Julia Julian Kai Karim Kate Kenji Kofi Laila Lars Laura Leah Leo Leon Liam Lina Lucas Lucia Luis Maja Malik Maria Marco Mariana
	Mark Marta Mateo Max Maya Mei Mia Michael Mina Mohamed Nadia Naomi Nathan Nia Nikolai Nina Noah Nora Olga Oliver Olivia Oscar Pablo
	Paula Pedro Priya Rafael Rahul Ravi Rosa Ryan Sara Sebastian Sofia Sophie Stefan Sven Tariq Theo Thomas Tomas Valentina Victor
	Wei Yara Yasmin Yuki Yusuf Zainab Zoe`)

var lastNames = strings.Fields(`
	Abbott Adeyemi Aguilar Ahmed Akhtar Alvarez Andersen Bakker Banerjee Barros Becker Bennett Bergstrom Bianchi Brooks Castillo
	Chandra Chen Costa Cruz Dahl Das Diaz Dubois Edwards Eriksen Esposito Evans Fernandes Fischer Fontaine Garcia Gomez Gonzalez
	Grant Gupta Hansen Hartmann Herrera Hoffmann Hughes Ibrahim Ito Ivanova Jensen Johansson Johnson Jung Kamau Kaur Keller Khan Kim
	Klein Kobayashi Kowalski Kumar Lambert Larsen Lee Lehmann Lopez Lund Martin Martinez Meyer Mendes Moreau Morales Moretti Muller
	Murphy Nakamura Nguyen Nielsen Novak Obi Okafor Oliveira Ortiz Osei Park Patel Perez Petrov Pham Popescu Quinn Ramirez Reddy
	Reyes Ricci Rodriguez Romano Rossi Sanchez Santos Sato Schmidt Schneider Shah Silva Singh Smith Sokolov Souza Suzuki Tanaka
	Taylor Thompson Torres Tran Vargas Vasquez Vogel Wagner Walker Wang Weber Williams Wilson Wong Yamamoto Yilmaz Young Zhang Zimmermann`)

// companyDomains are fictional businesses behind "Other" mailboxes, with
// the hosting their MX records point to.
var companyDomains = []struct{ domain, hosting string }{
	{"northwind-labs.com", "Google Workspace"}, {"bergconsulting.de", "Google Workspace"}, {"rahmanlabs.io", "Fastmail"},
	{"lattice-labs.io", "Google Workspace"}, {"haddadstudio.com", "Google Workspace"}, {"brightpath.co", "Microsoft 365"},
	{"kestrel.dev", "Google Workspace"}, {"meridian-analytics.com", "Microsoft 365"}, {"bluefin.io", "Google Workspace"},
	{"oakandiron.com", "Microsoft 365"}, {"pixelharbor.com", "Google Workspace"}, {"quillworks.co", "Fastmail"},
	{"redwoodcap.com", "Microsoft 365"}, {"saltmarsh.studio", "Google Workspace"}, {"tidewater.io", "Google Workspace"},
	{"umbra-systems.com", "Microsoft 365"}, {"vantagepoint.de", "Microsoft 365"}, {"willowtree.agency", "Google Workspace"},
	{"yellowbirch.com", "Proton"}, {"zenithworks.io", "Google Workspace"}, {"acorn-design.co.uk", "Microsoft 365"},
	{"basalt.engineering", "Google Workspace"}, {"cinder.app", "Google Workspace"}, {"driftwood.agency", "Fastmail"},
	{"ember-studio.fr", "Google Workspace"}, {"fjordsoft.no", "Microsoft 365"}, {"granitebay.com", "Google Workspace"},
	{"helix-bio.com", "Microsoft 365"}, {"ironclad-legal.com", "Microsoft 365"}, {"juniper-health.org", "Google Workspace"},
	{"kiteline.io", "Google Workspace"}, {"lumenhq.com", "Google Workspace"}, {"mosaic.health", "Microsoft 365"},
	{"nimbusdata.io", "Google Workspace"}, {"orbitlabs.dev", "Google Workspace"}, {"parallel.coop", "Proton"},
	{"quartzhq.com", "Google Workspace"}, {"riverstone.edu", "Microsoft 365"}, {"sableworks.com", "Google Workspace"},
	{"terrace.vc", "Google Workspace"}, {"uplift-ngo.org", "Google Workspace"}, {"vectorfield.ai", "Google Workspace"},
	{"wavecrest.io", "Microsoft 365"}, {"xenon-labs.com", "Google Workspace"}, {"yardstick.co", "Google Workspace"},
	{"zephyr-air.com", "Microsoft 365"}, {"fastmail.com", "Fastmail"}, {"proton.me", "Proton"}, {"gmx.de", ""}, {"web.de", ""},
}

var (
	outlookDomains = []string{"outlook.com", "outlook.com", "outlook.com", "outlook.com", "outlook.com", "outlook.com", "hotmail.com", "hotmail.com", "hotmail.com", "live.com"}
	yahooDomains   = []string{"yahoo.com", "yahoo.com", "yahoo.com", "yahoo.com", "yahoo.com", "yahoo.com", "yahoo.com", "yahoo.com", "aol.com", "ymail.com"}
	icloudDomains  = []string{"icloud.com", "icloud.com", "icloud.com", "icloud.com", "me.com"}
)

var countries = []struct{ code, tz string }{
	{"US", "America/New_York"}, {"US", "America/Chicago"}, {"US", "America/Los_Angeles"}, {"US", "America/Denver"},
	{"GB", "Europe/London"}, {"DE", "Europe/Berlin"}, {"FR", "Europe/Paris"}, {"NL", "Europe/Amsterdam"}, {"ES", "Europe/Madrid"},
	{"IT", "Europe/Rome"}, {"SE", "Europe/Stockholm"}, {"PL", "Europe/Warsaw"}, {"CA", "America/Toronto"}, {"BR", "America/Sao_Paulo"},
	{"MX", "America/Mexico_City"}, {"IN", "Asia/Kolkata"}, {"JP", "Asia/Tokyo"}, {"SG", "Asia/Singapore"}, {"AU", "Australia/Sydney"},
	{"NG", "Africa/Lagos"}, {"ZA", "Africa/Johannesburg"}, {"KE", "Africa/Nairobi"},
}
