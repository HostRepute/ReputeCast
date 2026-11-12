from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.scope_summary import ScopeSummary
  from ..models.team_role_ref import TeamRoleRef





T = TypeVar("T", bound="TeamInvitation")



@_attrs_define
class TeamInvitation:
    """ 
        Attributes:
            created_at (datetime.datetime):
            email (str):
            expired (bool):
            expires_at (datetime.datetime):
            id (str):
            invited_by (str):
            message (str):
            role (TeamRoleRef):
            scope (ScopeSummary):
            sent_via (str):
            last_sent_at (datetime.datetime | Unset):
     """

    created_at: datetime.datetime
    email: str
    expired: bool
    expires_at: datetime.datetime
    id: str
    invited_by: str
    message: str
    role: TeamRoleRef
    scope: ScopeSummary
    sent_via: str
    last_sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.scope_summary import ScopeSummary # noqa: PLC0415
        from ..models.team_role_ref import TeamRoleRef # noqa: PLC0415
        created_at = self.created_at.isoformat()

        email = self.email

        expired = self.expired

        expires_at = self.expires_at.isoformat()

        id = self.id

        invited_by = self.invited_by

        message = self.message

        role = self.role.to_dict()

        scope = self.scope.to_dict()

        sent_via = self.sent_via

        last_sent_at: str | Unset = UNSET
        if not isinstance(self.last_sent_at, Unset):
            last_sent_at = self.last_sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "email": email,
            "expired": expired,
            "expiresAt": expires_at,
            "id": id,
            "invitedBy": invited_by,
            "message": message,
            "role": role,
            "scope": scope,
            "sentVia": sent_via,
        })
        if last_sent_at is not UNSET:
            field_dict["lastSentAt"] = last_sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.scope_summary import ScopeSummary # noqa: PLC0415
        from ..models.team_role_ref import TeamRoleRef # noqa: PLC0415
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        email = d.pop("email")

        expired = d.pop("expired")

        expires_at = datetime.datetime.fromisoformat(d.pop("expiresAt"))




        id = d.pop("id")

        invited_by = d.pop("invitedBy")

        message = d.pop("message")

        role = TeamRoleRef.from_dict(d.pop("role"))




        scope = ScopeSummary.from_dict(d.pop("scope"))




        sent_via = d.pop("sentVia")

        _last_sent_at = d.pop("lastSentAt", UNSET)
        last_sent_at: datetime.datetime | Unset
        if isinstance(_last_sent_at,  Unset):
            last_sent_at = UNSET
        else:
            last_sent_at = datetime.datetime.fromisoformat(_last_sent_at)




        team_invitation = cls(
            created_at=created_at,
            email=email,
            expired=expired,
            expires_at=expires_at,
            id=id,
            invited_by=invited_by,
            message=message,
            role=role,
            scope=scope,
            sent_via=sent_via,
            last_sent_at=last_sent_at,
        )

        return team_invitation

