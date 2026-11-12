from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.poll_breakdown_row import PollBreakdownRow





T = TypeVar("T", bound="PollBreakdowns")



@_attrs_define
class PollBreakdowns:
    """ 
        Attributes:
            languages (list[PollBreakdownRow]):
            lists (list[PollBreakdownRow]):
            tiers (list[PollBreakdownRow]):
     """

    languages: list[PollBreakdownRow]
    lists: list[PollBreakdownRow]
    tiers: list[PollBreakdownRow]





    def to_dict(self) -> dict[str, Any]:
        from ..models.poll_breakdown_row import PollBreakdownRow # noqa: PLC0415
        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "languages": languages,
            "lists": lists,
            "tiers": tiers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.poll_breakdown_row import PollBreakdownRow # noqa: PLC0415
        d = dict(src_dict)
        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = PollBreakdownRow.from_dict(languages_item_data)



            languages.append(languages_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = PollBreakdownRow.from_dict(lists_item_data)



            lists.append(lists_item)


        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = PollBreakdownRow.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        poll_breakdowns = cls(
            languages=languages,
            lists=lists,
            tiers=tiers,
        )

        return poll_breakdowns

