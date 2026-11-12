from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationMilestoneOption")



@_attrs_define
class AutomationMilestoneOption:
    """ 
        Attributes:
            count (int):  Example: 3.
            id (str):
            list_name (str): The list of the milestone's referral program Example: Weekly Digest.
            name (str):  Example: 3 referrals · The Acme Field Guide.
            program_id (str):
            reward (str):  Example: The Acme Field Guide.
     """

    count: int
    id: str
    list_name: str
    name: str
    program_id: str
    reward: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        id = self.id

        list_name = self.list_name

        name = self.name

        program_id = self.program_id

        reward = self.reward


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "id": id,
            "listName": list_name,
            "name": name,
            "programId": program_id,
            "reward": reward,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        id = d.pop("id")

        list_name = d.pop("listName")

        name = d.pop("name")

        program_id = d.pop("programId")

        reward = d.pop("reward")

        automation_milestone_option = cls(
            count=count,
            id=id,
            list_name=list_name,
            name=name,
            program_id=program_id,
            reward=reward,
        )

        return automation_milestone_option

