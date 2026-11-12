from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ReportBacklogProvider")



@_attrs_define
class ReportBacklogProvider:
    """ 
        Attributes:
            feed (str): How its events arrive: SNS, webhook, bounce mailbox
            name (str):
            provider_id (str):
            short (str):
            since (datetime.datetime):
            waiting (int):
     """

    feed: str
    name: str
    provider_id: str
    short: str
    since: datetime.datetime
    waiting: int





    def to_dict(self) -> dict[str, Any]:
        feed = self.feed

        name = self.name

        provider_id = self.provider_id

        short = self.short

        since = self.since.isoformat()

        waiting = self.waiting


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "feed": feed,
            "name": name,
            "providerId": provider_id,
            "short": short,
            "since": since,
            "waiting": waiting,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        feed = d.pop("feed")

        name = d.pop("name")

        provider_id = d.pop("providerId")

        short = d.pop("short")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        waiting = d.pop("waiting")

        report_backlog_provider = cls(
            feed=feed,
            name=name,
            provider_id=provider_id,
            short=short,
            since=since,
            waiting=waiting,
        )

        return report_backlog_provider

