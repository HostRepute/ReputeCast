from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.report_backlog_provider import ReportBacklogProvider





T = TypeVar("T", bound="ReportIngestBacklog")



@_attrs_define
class ReportIngestBacklog:
    """ 
        Attributes:
            current (list[str]):
            minutes (int):
            providers (list[ReportBacklogProvider]):
            since (datetime.datetime):
     """

    current: list[str]
    minutes: int
    providers: list[ReportBacklogProvider]
    since: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_backlog_provider import ReportBacklogProvider # noqa: PLC0415
        current = self.current



        minutes = self.minutes

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "current": current,
            "minutes": minutes,
            "providers": providers,
            "since": since,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_backlog_provider import ReportBacklogProvider # noqa: PLC0415
        d = dict(src_dict)
        current = cast(list[str], d.pop("current"))


        minutes = d.pop("minutes")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = ReportBacklogProvider.from_dict(providers_item_data)



            providers.append(providers_item)


        since = datetime.datetime.fromisoformat(d.pop("since"))




        report_ingest_backlog = cls(
            current=current,
            minutes=minutes,
            providers=providers,
            since=since,
        )

        return report_ingest_backlog

