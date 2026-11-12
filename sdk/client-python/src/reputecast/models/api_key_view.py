from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.api_key_view_kind import APIKeyViewKind
from ..models.api_key_view_status import APIKeyViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.api_key_creator import APIKeyCreator
  from ..models.api_key_rejections import APIKeyRejections
  from ..models.api_key_transactional import APIKeyTransactional
  from ..models.scope_summary import ScopeSummary





T = TypeVar("T", bound="APIKeyView")



@_attrs_define
class APIKeyView:
    """ 
        Attributes:
            allowed_ips (list[str]): Addresses and networks the key may be called from (203.0.113.7, 203.0.113.0/24); empty
                for any address
            created_at (datetime.datetime):
            created_by (APIKeyCreator):
            description (str): Client label for agents (Claude Desktop)
            id (str):
            kind (APIKeyViewKind):
            masked (str):  Example: rc_live_8f2c…a91d.
            name (str):
            permissions (list[str]):
            scope (ScopeSummary):
            sensitive (list[str]): The ⚑ permissions among them
            status (APIKeyViewStatus):
            expires_at (datetime.datetime | Unset):
            last_call (str | Unset):
            last_used_at (datetime.datetime | Unset):
            last_used_client (str | Unset):
            last_used_ip (str | Unset):
            rejections (APIKeyRejections | Unset):
            transactional (APIKeyTransactional | Unset):
     """

    allowed_ips: list[str]
    created_at: datetime.datetime
    created_by: APIKeyCreator
    description: str
    id: str
    kind: APIKeyViewKind
    masked: str
    name: str
    permissions: list[str]
    scope: ScopeSummary
    sensitive: list[str]
    status: APIKeyViewStatus
    expires_at: datetime.datetime | Unset = UNSET
    last_call: str | Unset = UNSET
    last_used_at: datetime.datetime | Unset = UNSET
    last_used_client: str | Unset = UNSET
    last_used_ip: str | Unset = UNSET
    rejections: APIKeyRejections | Unset = UNSET
    transactional: APIKeyTransactional | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.api_key_creator import APIKeyCreator # noqa: PLC0415
        from ..models.api_key_rejections import APIKeyRejections # noqa: PLC0415
        from ..models.api_key_transactional import APIKeyTransactional # noqa: PLC0415
        from ..models.scope_summary import ScopeSummary # noqa: PLC0415
        allowed_ips = self.allowed_ips



        created_at = self.created_at.isoformat()

        created_by = self.created_by.to_dict()

        description = self.description

        id = self.id

        kind = self.kind.value

        masked = self.masked

        name = self.name

        permissions = self.permissions



        scope = self.scope.to_dict()

        sensitive = self.sensitive



        status = self.status.value

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        last_call = self.last_call

        last_used_at: str | Unset = UNSET
        if not isinstance(self.last_used_at, Unset):
            last_used_at = self.last_used_at.isoformat()

        last_used_client = self.last_used_client

        last_used_ip = self.last_used_ip

        rejections: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rejections, Unset):
            rejections = self.rejections.to_dict()

        transactional: dict[str, Any] | Unset = UNSET
        if not isinstance(self.transactional, Unset):
            transactional = self.transactional.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allowedIps": allowed_ips,
            "createdAt": created_at,
            "createdBy": created_by,
            "description": description,
            "id": id,
            "kind": kind,
            "masked": masked,
            "name": name,
            "permissions": permissions,
            "scope": scope,
            "sensitive": sensitive,
            "status": status,
        })
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if last_call is not UNSET:
            field_dict["lastCall"] = last_call
        if last_used_at is not UNSET:
            field_dict["lastUsedAt"] = last_used_at
        if last_used_client is not UNSET:
            field_dict["lastUsedClient"] = last_used_client
        if last_used_ip is not UNSET:
            field_dict["lastUsedIp"] = last_used_ip
        if rejections is not UNSET:
            field_dict["rejections"] = rejections
        if transactional is not UNSET:
            field_dict["transactional"] = transactional

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.api_key_creator import APIKeyCreator # noqa: PLC0415
        from ..models.api_key_rejections import APIKeyRejections # noqa: PLC0415
        from ..models.api_key_transactional import APIKeyTransactional # noqa: PLC0415
        from ..models.scope_summary import ScopeSummary # noqa: PLC0415
        d = dict(src_dict)
        allowed_ips = cast(list[str], d.pop("allowedIps"))


        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = APIKeyCreator.from_dict(d.pop("createdBy"))




        description = d.pop("description")

        id = d.pop("id")

        kind = APIKeyViewKind(d.pop("kind"))




        masked = d.pop("masked")

        name = d.pop("name")

        permissions = cast(list[str], d.pop("permissions"))


        scope = ScopeSummary.from_dict(d.pop("scope"))




        sensitive = cast(list[str], d.pop("sensitive"))


        status = APIKeyViewStatus(d.pop("status"))




        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        last_call = d.pop("lastCall", UNSET)

        _last_used_at = d.pop("lastUsedAt", UNSET)
        last_used_at: datetime.datetime | Unset
        if isinstance(_last_used_at,  Unset):
            last_used_at = UNSET
        else:
            last_used_at = datetime.datetime.fromisoformat(_last_used_at)




        last_used_client = d.pop("lastUsedClient", UNSET)

        last_used_ip = d.pop("lastUsedIp", UNSET)

        _rejections = d.pop("rejections", UNSET)
        rejections: APIKeyRejections | Unset
        if isinstance(_rejections,  Unset):
            rejections = UNSET
        else:
            rejections = APIKeyRejections.from_dict(_rejections)




        _transactional = d.pop("transactional", UNSET)
        transactional: APIKeyTransactional | Unset
        if isinstance(_transactional,  Unset):
            transactional = UNSET
        else:
            transactional = APIKeyTransactional.from_dict(_transactional)




        api_key_view = cls(
            allowed_ips=allowed_ips,
            created_at=created_at,
            created_by=created_by,
            description=description,
            id=id,
            kind=kind,
            masked=masked,
            name=name,
            permissions=permissions,
            scope=scope,
            sensitive=sensitive,
            status=status,
            expires_at=expires_at,
            last_call=last_call,
            last_used_at=last_used_at,
            last_used_client=last_used_client,
            last_used_ip=last_used_ip,
            rejections=rejections,
            transactional=transactional,
        )

        return api_key_view

