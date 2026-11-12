from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PollOverTime")



@_attrs_define
class PollOverTime:
    """ 
        Attributes:
            after (int):
            cumulative (list[int]):
            first_six_percent (float):
            since (datetime.datetime | Unset):
     """

    after: int
    cumulative: list[int]
    first_six_percent: float
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        after = self.after

        cumulative = self.cumulative



        first_six_percent = self.first_six_percent

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "after": after,
            "cumulative": cumulative,
            "firstSixPercent": first_six_percent,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        after = d.pop("after")

        cumulative = cast(list[int], d.pop("cumulative"))


        first_six_percent = d.pop("firstSixPercent")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        poll_over_time = cls(
            after=after,
            cumulative=cumulative,
            first_six_percent=first_six_percent,
            since=since,
        )

        return poll_over_time

