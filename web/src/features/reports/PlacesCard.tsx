import type { Schemas } from '@/api/client'
import { ButtonAnchor } from '@/components/ui/Button'
import { TableScroll, Th } from '@/components/ui/Table'
import { countryName } from '@/features/sms/common'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { ShareCell } from './DevicesCard'
import { m } from './messages'

type Breakdown = Schemas['ReportBreakdown']

/** The configuration guide's GeoIP setting (docs/deploy/configuration.md). */
const GEOIP_GUIDE = 'https://docs.reputecast.com/self-hosting/configuration/'

/**
 * "Countries" (Report board, M57 G-12): human opens per country and the top
 * cities, from the reader's address at their first human open (the address
 * isn't kept). Read-only links get no cities, so that pane goes.
 */
export function PlacesCard({ b }: { b: Breakdown }) {
  const t = useT()
  const located = b.countries.length > 0
  const body = located ? null : !b.geoip ? (
    <div className="flex flex-col items-start gap-2.5 px-4.5 py-3.5">
      <p className="k m-0">{t(m.geoOff)}</p>
      <ButtonAnchor size="sm" href={GEOIP_GUIDE} icon="external">
        {t(m.geoHow)}
        <span className="vh"> {t(m.newTab)}</span>
      </ButtonAnchor>
    </div>
  ) : (
    <p className="k m-0 px-4.5 py-3.5">{b.clients.length ? t(m.geoNone) : t(m.noHumanOpens)}</p>
  )
  return (
    <section className="card flex min-w-0 flex-col" aria-labelledby="rep-geo-h">
      <div className="sec">
        <h2 className="h2" id="rep-geo-h">
          {t(m.placesTitle)}
        </h2>
        <span className="k">{t(m.placesCaption)}</span>
      </div>
      {body ?? (
        <>
          <div className="rep-panes rounded-none">
            <div className="flex min-w-0 flex-[1_1_360px] flex-col px-2.5 pb-1">
              <TableScroll label={t(m.countriesTable)}>
                <table className="tbl rep-tbl">
                  <caption className="vh">{t(m.countriesTable)}</caption>
                  <thead>
                    <tr>
                      <Th>{t(m.colCountry)}</Th>
                      <Th align="right">{t(m.humanOpens)}</Th>
                      <Th>{t(m.colShareOpens)}</Th>
                      <Th align="right">{t(m.humanClicks)}</Th>
                    </tr>
                  </thead>
                  <tbody>
                    {b.countries.map((r) => (
                      <tr key={r.key}>
                        <th scope="row">
                          {r.key === 'other' ? t(m.otherCountries) : countryName(r.key)}
                          {r.key === 'other' && r.places ? <span className="k block font-medium">{t(m.countriesCount, { n: r.places })}</span> : null}
                        </th>
                        <td className="r n">{count(r.humanOpens)}</td>
                        <ShareCell share={r.share} />
                        <td className="r n">{count(r.clicks)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </TableScroll>
            </div>
            {b.cities.length ? (
              <div className="flex min-w-0 flex-[1_1_360px] flex-col px-2.5 pb-1">
                <h3 className="h3 m-0 px-2 pt-3.5">{t(m.topCities)}</h3>
                <TableScroll label={t(m.citiesTable)}>
                  <table className="tbl rep-tbl">
                    <caption className="vh">{t(m.citiesTable)}</caption>
                    <thead>
                      <tr>
                        <Th>{t(m.colCity)}</Th>
                        <Th align="right">{t(m.humanOpens)}</Th>
                        <Th align="right">{t(m.humanClicks)}</Th>
                      </tr>
                    </thead>
                    <tbody>
                      {b.cities.map((r) => (
                        <tr key={r.key}>
                          <th scope="row">
                            {r.key === 'other' ? t(m.otherCities) : r.label}
                            <span className="k block font-medium">
                              {r.key === 'other' ? t(m.citiesCount, { n: r.places ?? 0 }) : r.country ? countryName(r.country) : null}
                            </span>
                          </th>
                          <td className="r n">{count(r.humanOpens)}</td>
                          <td className="r n">{count(r.clicks)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </TableScroll>
              </div>
            ) : null}
          </div>
          <div className="k flex flex-col gap-1 border-t border-line-2 px-4.5 py-2.5">
            <span>{t(m.placesShares, { count: count(b.located) })}</span>
            {b.cities.length ? <span>{t(m.citiesFold)}</span> : null}
          </div>
        </>
      )}
    </section>
  )
}
