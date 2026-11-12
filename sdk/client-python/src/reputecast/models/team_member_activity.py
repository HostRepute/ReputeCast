from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.team_member_activity_result import TeamMemberActivityResult
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TeamMemberActivity")



@_attrs_define
class TeamMemberActivity:
    """ 
        Attributes:
            action (str):
            id (str):
            occurred_at (datetime.datetime):
            result (TeamMemberActivityResult):
            target (str):
            reason (str | Unset):
     """

    action: str
    id: str
    occurred_at: datetime.datetime
    result: TeamMemberActivityResult
    target: str
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action

        id = self.id

        occurred_at = self.occurred_at.isoformat()

        result = self.result.value

        target = self.target

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "id": id,
            "occurredAt": occurred_at,
            "result": result,
            "target": target,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = d.pop("action")

        id = d.pop("id")

        occurred_at = datetime.datetime.fromisoformat(d.pop("occurredAt"))




        result = TeamMemberActivityResult(d.pop("result"))




        target = d.pop("target")

        reason = d.pop("reason", UNSET)

        team_member_activity = cls(
            action=action,
            id=id,
            occurred_at=occurred_at,
            result=result,
            target=target,
            reason=reason,
        )

        return team_member_activity

