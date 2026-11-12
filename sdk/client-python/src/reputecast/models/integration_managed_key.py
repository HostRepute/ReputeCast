from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_managed_key_owner_status import IntegrationManagedKeyOwnerStatus
from ..models.integration_managed_key_status import IntegrationManagedKeyStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.integration_person import IntegrationPerson
  from ..models.integration_workflow_ref import IntegrationWorkflowRef
  from ..models.scope_summary import ScopeSummary





T = TypeVar("T", bound="IntegrationManagedKey")



@_attrs_define
class IntegrationManagedKey:
    """ 
        Attributes:
            calls7d (int):
            connection_id (str):
            created_at (datetime.datetime):
            id (str):
            kind (str):
            label (str):
            masked (str):  Example: rc_live_5d1e…7b20.
            name (str):
            owner (IntegrationPerson):
            owner_status (IntegrationManagedKeyOwnerStatus):
            permissions (list[str]): As held: implied view permissions left out
            scope (ScopeSummary):
            sensitive (list[str]):
            status (IntegrationManagedKeyStatus):
            workflows (list[IntegrationWorkflowRef]):
            expires_at (datetime.datetime | Unset):
            grace_until (datetime.datetime | Unset): A rotated key keeps working until then
            last_used_at (datetime.datetime | Unset):
            last_used_client (str | Unset):
            last_used_ip (str | Unset):
            owner_left_at (datetime.datetime | Unset):
            revoked_at (datetime.datetime | Unset):
     """

    calls7d: int
    connection_id: str
    created_at: datetime.datetime
    id: str
    kind: str
    label: str
    masked: str
    name: str
    owner: IntegrationPerson
    owner_status: IntegrationManagedKeyOwnerStatus
    permissions: list[str]
    scope: ScopeSummary
    sensitive: list[str]
    status: IntegrationManagedKeyStatus
    workflows: list[IntegrationWorkflowRef]
    expires_at: datetime.datetime | Unset = UNSET
    grace_until: datetime.datetime | Unset = UNSET
    last_used_at: datetime.datetime | Unset = UNSET
    last_used_client: str | Unset = UNSET
    last_used_ip: str | Unset = UNSET
    owner_left_at: datetime.datetime | Unset = UNSET
    revoked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_person import IntegrationPerson # noqa: PLC0415
        from ..models.integration_workflow_ref import IntegrationWorkflowRef # noqa: PLC0415
        from ..models.scope_summary import ScopeSummary # noqa: PLC0415
        calls7d = self.calls7d

        connection_id = self.connection_id

        created_at = self.created_at.isoformat()

        id = self.id

        kind = self.kind

        label = self.label

        masked = self.masked

        name = self.name

        owner = self.owner.to_dict()

        owner_status = self.owner_status.value

        permissions = self.permissions



        scope = self.scope.to_dict()

        sensitive = self.sensitive



        status = self.status.value

        workflows = []
        for workflows_item_data in self.workflows:
            workflows_item = workflows_item_data.to_dict()
            workflows.append(workflows_item)



        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        grace_until: str | Unset = UNSET
        if not isinstance(self.grace_until, Unset):
            grace_until = self.grace_until.isoformat()

        last_used_at: str | Unset = UNSET
        if not isinstance(self.last_used_at, Unset):
            last_used_at = self.last_used_at.isoformat()

        last_used_client = self.last_used_client

        last_used_ip = self.last_used_ip

        owner_left_at: str | Unset = UNSET
        if not isinstance(self.owner_left_at, Unset):
            owner_left_at = self.owner_left_at.isoformat()

        revoked_at: str | Unset = UNSET
        if not isinstance(self.revoked_at, Unset):
            revoked_at = self.revoked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "calls7d": calls7d,
            "connectionId": connection_id,
            "createdAt": created_at,
            "id": id,
            "kind": kind,
            "label": label,
            "masked": masked,
            "name": name,
            "owner": owner,
            "ownerStatus": owner_status,
            "permissions": permissions,
            "scope": scope,
            "sensitive": sensitive,
            "status": status,
            "workflows": workflows,
        })
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if grace_until is not UNSET:
            field_dict["graceUntil"] = grace_until
        if last_used_at is not UNSET:
            field_dict["lastUsedAt"] = last_used_at
        if last_used_client is not UNSET:
            field_dict["lastUsedClient"] = last_used_client
        if last_used_ip is not UNSET:
            field_dict["lastUsedIp"] = last_used_ip
        if owner_left_at is not UNSET:
            field_dict["ownerLeftAt"] = owner_left_at
        if revoked_at is not UNSET:
            field_dict["revokedAt"] = revoked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_person import IntegrationPerson # noqa: PLC0415
        from ..models.integration_workflow_ref import IntegrationWorkflowRef # noqa: PLC0415
        from ..models.scope_summary import ScopeSummary # noqa: PLC0415
        d = dict(src_dict)
        calls7d = d.pop("calls7d")

        connection_id = d.pop("connectionId")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        id = d.pop("id")

        kind = d.pop("kind")

        label = d.pop("label")

        masked = d.pop("masked")

        name = d.pop("name")

        owner = IntegrationPerson.from_dict(d.pop("owner"))




        owner_status = IntegrationManagedKeyOwnerStatus(d.pop("ownerStatus"))




        permissions = cast(list[str], d.pop("permissions"))


        scope = ScopeSummary.from_dict(d.pop("scope"))




        sensitive = cast(list[str], d.pop("sensitive"))


        status = IntegrationManagedKeyStatus(d.pop("status"))




        workflows = []
        _workflows = d.pop("workflows")
        for workflows_item_data in (_workflows):
            workflows_item = IntegrationWorkflowRef.from_dict(workflows_item_data)



            workflows.append(workflows_item)


        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _grace_until = d.pop("graceUntil", UNSET)
        grace_until: datetime.datetime | Unset
        if isinstance(_grace_until,  Unset):
            grace_until = UNSET
        else:
            grace_until = datetime.datetime.fromisoformat(_grace_until)




        _last_used_at = d.pop("lastUsedAt", UNSET)
        last_used_at: datetime.datetime | Unset
        if isinstance(_last_used_at,  Unset):
            last_used_at = UNSET
        else:
            last_used_at = datetime.datetime.fromisoformat(_last_used_at)




        last_used_client = d.pop("lastUsedClient", UNSET)

        last_used_ip = d.pop("lastUsedIp", UNSET)

        _owner_left_at = d.pop("ownerLeftAt", UNSET)
        owner_left_at: datetime.datetime | Unset
        if isinstance(_owner_left_at,  Unset):
            owner_left_at = UNSET
        else:
            owner_left_at = datetime.datetime.fromisoformat(_owner_left_at)




        _revoked_at = d.pop("revokedAt", UNSET)
        revoked_at: datetime.datetime | Unset
        if isinstance(_revoked_at,  Unset):
            revoked_at = UNSET
        else:
            revoked_at = datetime.datetime.fromisoformat(_revoked_at)




        integration_managed_key = cls(
            calls7d=calls7d,
            connection_id=connection_id,
            created_at=created_at,
            id=id,
            kind=kind,
            label=label,
            masked=masked,
            name=name,
            owner=owner,
            owner_status=owner_status,
            permissions=permissions,
            scope=scope,
            sensitive=sensitive,
            status=status,
            workflows=workflows,
            expires_at=expires_at,
            grace_until=grace_until,
            last_used_at=last_used_at,
            last_used_client=last_used_client,
            last_used_ip=last_used_ip,
            owner_left_at=owner_left_at,
            revoked_at=revoked_at,
        )

        return integration_managed_key

