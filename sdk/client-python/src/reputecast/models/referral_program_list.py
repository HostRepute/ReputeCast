from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.grow_helper_sets import GrowHelperSets
  from ..models.referral_list_option import ReferralListOption
  from ..models.referral_program_summary import ReferralProgramSummary





T = TypeVar("T", bound="ReferralProgramList")



@_attrs_define
class ReferralProgramList:
    """ 
        Attributes:
            helpers (GrowHelperSets):
            items (list[ReferralProgramSummary]):
            lists (list[ReferralListOption]):
     """

    helpers: GrowHelperSets
    items: list[ReferralProgramSummary]
    lists: list[ReferralListOption]





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_list_option import ReferralListOption # noqa: PLC0415
        from ..models.referral_program_summary import ReferralProgramSummary # noqa: PLC0415
        helpers = self.helpers.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "helpers": helpers,
            "items": items,
            "lists": lists,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_helper_sets import GrowHelperSets # noqa: PLC0415
        from ..models.referral_list_option import ReferralListOption # noqa: PLC0415
        from ..models.referral_program_summary import ReferralProgramSummary # noqa: PLC0415
        d = dict(src_dict)
        helpers = GrowHelperSets.from_dict(d.pop("helpers"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ReferralProgramSummary.from_dict(items_item_data)



            items.append(items_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ReferralListOption.from_dict(lists_item_data)



            lists.append(lists_item)


        referral_program_list = cls(
            helpers=helpers,
            items=items,
            lists=lists,
        )

        return referral_program_list

