from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.team_invitation import TeamInvitation
  from ..models.team_jit_hint import TeamJITHint





T = TypeVar("T", bound="TeamInvitationList")



@_attrs_define
class TeamInvitationList:
    """ 
        Attributes:
            can_invite (bool):
            default_days (int):
            items (list[TeamInvitation]):
            jit (list[TeamJITHint]):
     """

    can_invite: bool
    default_days: int
    items: list[TeamInvitation]
    jit: list[TeamJITHint]





    def to_dict(self) -> dict[str, Any]:
        from ..models.team_invitation import TeamInvitation # noqa: PLC0415
        from ..models.team_jit_hint import TeamJITHint # noqa: PLC0415
        can_invite = self.can_invite

        default_days = self.default_days

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        jit = []
        for jit_item_data in self.jit:
            jit_item = jit_item_data.to_dict()
            jit.append(jit_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canInvite": can_invite,
            "defaultDays": default_days,
            "items": items,
            "jit": jit,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.team_invitation import TeamInvitation # noqa: PLC0415
        from ..models.team_jit_hint import TeamJITHint # noqa: PLC0415
        d = dict(src_dict)
        can_invite = d.pop("canInvite")

        default_days = d.pop("defaultDays")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = TeamInvitation.from_dict(items_item_data)



            items.append(items_item)


        jit = []
        _jit = d.pop("jit")
        for jit_item_data in (_jit):
            jit_item = TeamJITHint.from_dict(jit_item_data)



            jit.append(jit_item)


        team_invitation_list = cls(
            can_invite=can_invite,
            default_days=default_days,
            items=items,
            jit=jit,
        )

        return team_invitation_list

