from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.growth_chart import GrowthChart
  from ..models.growth_cohort import GrowthCohort
  from ..models.growth_departures import GrowthDepartures
  from ..models.growth_import_item import GrowthImportItem
  from ..models.growth_list_ref import GrowthListRef
  from ..models.growth_period import GrowthPeriod
  from ..models.growth_rates import GrowthRates
  from ..models.growth_referrers import GrowthReferrers
  from ..models.growth_source import GrowthSource
  from ..models.growth_source_row import GrowthSourceRow
  from ..models.growth_tiles import GrowthTiles
  from ..models.growth_totals import GrowthTotals
  from ..models.growth_utm import GrowthUTM





T = TypeVar("T", bound="GrowthPage")



@_attrs_define
class GrowthPage:
    """ 
        Attributes:
            chart (GrowthChart):
            cohort_source (str):
            cohort_sources (list[GrowthSource]):
            cohorts (list[GrowthCohort]):
            contacts (int):
            departures (GrowthDepartures):
            flag_factor (float):
            imports (list[GrowthImportItem]): Every finished import, for the only-imports state
            include_imports (bool):
            lists (list[GrowthListRef]):
            members (bool):
            min_contacts (int):
            min_delivered (int):
            only_imports (bool):
            period (GrowthPeriod):
            referrers (GrowthReferrers):
            sources (list[GrowthSourceRow]):
            tiles (GrowthTiles):
            total (GrowthTotals):
            utm (GrowthUTM):
            workspace (GrowthRates):
            list_ (GrowthListRef | Unset):
     """

    chart: GrowthChart
    cohort_source: str
    cohort_sources: list[GrowthSource]
    cohorts: list[GrowthCohort]
    contacts: int
    departures: GrowthDepartures
    flag_factor: float
    imports: list[GrowthImportItem]
    include_imports: bool
    lists: list[GrowthListRef]
    members: bool
    min_contacts: int
    min_delivered: int
    only_imports: bool
    period: GrowthPeriod
    referrers: GrowthReferrers
    sources: list[GrowthSourceRow]
    tiles: GrowthTiles
    total: GrowthTotals
    utm: GrowthUTM
    workspace: GrowthRates
    list_: GrowthListRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_chart import GrowthChart # noqa: PLC0415
        from ..models.growth_cohort import GrowthCohort # noqa: PLC0415
        from ..models.growth_departures import GrowthDepartures # noqa: PLC0415
        from ..models.growth_import_item import GrowthImportItem # noqa: PLC0415
        from ..models.growth_list_ref import GrowthListRef # noqa: PLC0415
        from ..models.growth_period import GrowthPeriod # noqa: PLC0415
        from ..models.growth_rates import GrowthRates # noqa: PLC0415
        from ..models.growth_referrers import GrowthReferrers # noqa: PLC0415
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        from ..models.growth_source_row import GrowthSourceRow # noqa: PLC0415
        from ..models.growth_tiles import GrowthTiles # noqa: PLC0415
        from ..models.growth_totals import GrowthTotals # noqa: PLC0415
        from ..models.growth_utm import GrowthUTM # noqa: PLC0415
        chart = self.chart.to_dict()

        cohort_source = self.cohort_source

        cohort_sources = []
        for cohort_sources_item_data in self.cohort_sources:
            cohort_sources_item = cohort_sources_item_data.to_dict()
            cohort_sources.append(cohort_sources_item)



        cohorts = []
        for cohorts_item_data in self.cohorts:
            cohorts_item = cohorts_item_data.to_dict()
            cohorts.append(cohorts_item)



        contacts = self.contacts

        departures = self.departures.to_dict()

        flag_factor = self.flag_factor

        imports = []
        for imports_item_data in self.imports:
            imports_item = imports_item_data.to_dict()
            imports.append(imports_item)



        include_imports = self.include_imports

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        members = self.members

        min_contacts = self.min_contacts

        min_delivered = self.min_delivered

        only_imports = self.only_imports

        period = self.period.to_dict()

        referrers = self.referrers.to_dict()

        sources = []
        for sources_item_data in self.sources:
            sources_item = sources_item_data.to_dict()
            sources.append(sources_item)



        tiles = self.tiles.to_dict()

        total = self.total.to_dict()

        utm = self.utm.to_dict()

        workspace = self.workspace.to_dict()

        list_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.list_, Unset):
            list_ = self.list_.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "chart": chart,
            "cohortSource": cohort_source,
            "cohortSources": cohort_sources,
            "cohorts": cohorts,
            "contacts": contacts,
            "departures": departures,
            "flagFactor": flag_factor,
            "imports": imports,
            "includeImports": include_imports,
            "lists": lists,
            "members": members,
            "minContacts": min_contacts,
            "minDelivered": min_delivered,
            "onlyImports": only_imports,
            "period": period,
            "referrers": referrers,
            "sources": sources,
            "tiles": tiles,
            "total": total,
            "utm": utm,
            "workspace": workspace,
        })
        if list_ is not UNSET:
            field_dict["list"] = list_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_chart import GrowthChart # noqa: PLC0415
        from ..models.growth_cohort import GrowthCohort # noqa: PLC0415
        from ..models.growth_departures import GrowthDepartures # noqa: PLC0415
        from ..models.growth_import_item import GrowthImportItem # noqa: PLC0415
        from ..models.growth_list_ref import GrowthListRef # noqa: PLC0415
        from ..models.growth_period import GrowthPeriod # noqa: PLC0415
        from ..models.growth_rates import GrowthRates # noqa: PLC0415
        from ..models.growth_referrers import GrowthReferrers # noqa: PLC0415
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        from ..models.growth_source_row import GrowthSourceRow # noqa: PLC0415
        from ..models.growth_tiles import GrowthTiles # noqa: PLC0415
        from ..models.growth_totals import GrowthTotals # noqa: PLC0415
        from ..models.growth_utm import GrowthUTM # noqa: PLC0415
        d = dict(src_dict)
        chart = GrowthChart.from_dict(d.pop("chart"))




        cohort_source = d.pop("cohortSource")

        cohort_sources = []
        _cohort_sources = d.pop("cohortSources")
        for cohort_sources_item_data in (_cohort_sources):
            cohort_sources_item = GrowthSource.from_dict(cohort_sources_item_data)



            cohort_sources.append(cohort_sources_item)


        cohorts = []
        _cohorts = d.pop("cohorts")
        for cohorts_item_data in (_cohorts):
            cohorts_item = GrowthCohort.from_dict(cohorts_item_data)



            cohorts.append(cohorts_item)


        contacts = d.pop("contacts")

        departures = GrowthDepartures.from_dict(d.pop("departures"))




        flag_factor = d.pop("flagFactor")

        imports = []
        _imports = d.pop("imports")
        for imports_item_data in (_imports):
            imports_item = GrowthImportItem.from_dict(imports_item_data)



            imports.append(imports_item)


        include_imports = d.pop("includeImports")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = GrowthListRef.from_dict(lists_item_data)



            lists.append(lists_item)


        members = d.pop("members")

        min_contacts = d.pop("minContacts")

        min_delivered = d.pop("minDelivered")

        only_imports = d.pop("onlyImports")

        period = GrowthPeriod.from_dict(d.pop("period"))




        referrers = GrowthReferrers.from_dict(d.pop("referrers"))




        sources = []
        _sources = d.pop("sources")
        for sources_item_data in (_sources):
            sources_item = GrowthSourceRow.from_dict(sources_item_data)



            sources.append(sources_item)


        tiles = GrowthTiles.from_dict(d.pop("tiles"))




        total = GrowthTotals.from_dict(d.pop("total"))




        utm = GrowthUTM.from_dict(d.pop("utm"))




        workspace = GrowthRates.from_dict(d.pop("workspace"))




        _list_ = d.pop("list", UNSET)
        list_: GrowthListRef | Unset
        if isinstance(_list_,  Unset):
            list_ = UNSET
        else:
            list_ = GrowthListRef.from_dict(_list_)




        growth_page = cls(
            chart=chart,
            cohort_source=cohort_source,
            cohort_sources=cohort_sources,
            cohorts=cohorts,
            contacts=contacts,
            departures=departures,
            flag_factor=flag_factor,
            imports=imports,
            include_imports=include_imports,
            lists=lists,
            members=members,
            min_contacts=min_contacts,
            min_delivered=min_delivered,
            only_imports=only_imports,
            period=period,
            referrers=referrers,
            sources=sources,
            tiles=tiles,
            total=total,
            utm=utm,
            workspace=workspace,
            list_=list_,
        )

        return growth_page

