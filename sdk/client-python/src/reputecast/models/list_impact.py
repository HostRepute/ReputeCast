from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.list_impact_item import ListImpactItem





T = TypeVar("T", bound="ListImpact")



@_attrs_define
class ListImpact:
    """ 
        Attributes:
            automations (list[ListImpactItem]):
            campaigns (list[ListImpactItem]):
            elsewhere (int):
            forms (list[ListImpactItem]):
            id (str):
            members (int):
            name (str):
            other_list_count (int):
            other_lists (list[str]):
     """

    automations: list[ListImpactItem]
    campaigns: list[ListImpactItem]
    elsewhere: int
    forms: list[ListImpactItem]
    id: str
    members: int
    name: str
    other_list_count: int
    other_lists: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.list_impact_item import ListImpactItem # noqa: PLC0415
        automations = []
        for automations_item_data in self.automations:
            automations_item = automations_item_data.to_dict()
            automations.append(automations_item)



        campaigns = []
        for campaigns_item_data in self.campaigns:
            campaigns_item = campaigns_item_data.to_dict()
            campaigns.append(campaigns_item)



        elsewhere = self.elsewhere

        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        id = self.id

        members = self.members

        name = self.name

        other_list_count = self.other_list_count

        other_lists = self.other_lists




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automations": automations,
            "campaigns": campaigns,
            "elsewhere": elsewhere,
            "forms": forms,
            "id": id,
            "members": members,
            "name": name,
            "otherListCount": other_list_count,
            "otherLists": other_lists,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.list_impact_item import ListImpactItem # noqa: PLC0415
        d = dict(src_dict)
        automations = []
        _automations = d.pop("automations")
        for automations_item_data in (_automations):
            automations_item = ListImpactItem.from_dict(automations_item_data)



            automations.append(automations_item)


        campaigns = []
        _campaigns = d.pop("campaigns")
        for campaigns_item_data in (_campaigns):
            campaigns_item = ListImpactItem.from_dict(campaigns_item_data)



            campaigns.append(campaigns_item)


        elsewhere = d.pop("elsewhere")

        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = ListImpactItem.from_dict(forms_item_data)



            forms.append(forms_item)


        id = d.pop("id")

        members = d.pop("members")

        name = d.pop("name")

        other_list_count = d.pop("otherListCount")

        other_lists = cast(list[str], d.pop("otherLists"))


        list_impact = cls(
            automations=automations,
            campaigns=campaigns,
            elsewhere=elsewhere,
            forms=forms,
            id=id,
            members=members,
            name=name,
            other_list_count=other_list_count,
            other_lists=other_lists,
        )

        return list_impact

