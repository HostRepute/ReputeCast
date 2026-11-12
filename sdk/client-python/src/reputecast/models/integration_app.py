from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_app_kind import IntegrationAppKind
from ..models.integration_app_status import IntegrationAppStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.integration_first_call import IntegrationFirstCall
  from ..models.integration_item_state import IntegrationItemState
  from ..models.integration_key_view import IntegrationKeyView
  from ..models.integration_managed_hook import IntegrationManagedHook
  from ..models.integration_numbers import IntegrationNumbers
  from ..models.integration_person import IntegrationPerson
  from ..models.integration_reachability import IntegrationReachability
  from ..models.integration_reason import IntegrationReason
  from ..models.integration_rejected_call import IntegrationRejectedCall
  from ..models.integration_workflow_row import IntegrationWorkflowRow





T = TypeVar("T", bound="IntegrationApp")



@_attrs_define
class IntegrationApp:
    """ 
        Attributes:
            actions (int):
            can_manage (bool):
            helpers (list[str]):
            instance_url (str):
            items (list[IntegrationItemState]):
            kind (IntegrationAppKind):
            numbers (IntegrationNumbers):
            reachability (IntegrationReachability):
            reasons (list[IntegrationReason]):
            receives (list[str]):
            rejected (list[IntegrationRejectedCall]):
            status (IntegrationAppStatus):
            triggers (int):
            webhooks (list[IntegrationManagedHook]):
            workflows (list[IntegrationWorkflowRow]):
            address (str | Unset):
            connected_at (datetime.datetime | Unset):
            connected_by (IntegrationPerson | Unset):
            connection_id (str | Unset):
            first_call (IntegrationFirstCall | Unset):
            key (IntegrationKeyView | Unset):
            label (str | Unset):
            old_key (IntegrationKeyView | Unset):
     """

    actions: int
    can_manage: bool
    helpers: list[str]
    instance_url: str
    items: list[IntegrationItemState]
    kind: IntegrationAppKind
    numbers: IntegrationNumbers
    reachability: IntegrationReachability
    reasons: list[IntegrationReason]
    receives: list[str]
    rejected: list[IntegrationRejectedCall]
    status: IntegrationAppStatus
    triggers: int
    webhooks: list[IntegrationManagedHook]
    workflows: list[IntegrationWorkflowRow]
    address: str | Unset = UNSET
    connected_at: datetime.datetime | Unset = UNSET
    connected_by: IntegrationPerson | Unset = UNSET
    connection_id: str | Unset = UNSET
    first_call: IntegrationFirstCall | Unset = UNSET
    key: IntegrationKeyView | Unset = UNSET
    label: str | Unset = UNSET
    old_key: IntegrationKeyView | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_first_call import IntegrationFirstCall # noqa: PLC0415
        from ..models.integration_item_state import IntegrationItemState # noqa: PLC0415
        from ..models.integration_key_view import IntegrationKeyView # noqa: PLC0415
        from ..models.integration_managed_hook import IntegrationManagedHook # noqa: PLC0415
        from ..models.integration_numbers import IntegrationNumbers # noqa: PLC0415
        from ..models.integration_person import IntegrationPerson # noqa: PLC0415
        from ..models.integration_reachability import IntegrationReachability # noqa: PLC0415
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        from ..models.integration_rejected_call import IntegrationRejectedCall # noqa: PLC0415
        from ..models.integration_workflow_row import IntegrationWorkflowRow # noqa: PLC0415
        actions = self.actions

        can_manage = self.can_manage

        helpers = self.helpers



        instance_url = self.instance_url

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        kind = self.kind.value

        numbers = self.numbers.to_dict()

        reachability = self.reachability.to_dict()

        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        receives = self.receives



        rejected = []
        for rejected_item_data in self.rejected:
            rejected_item = rejected_item_data.to_dict()
            rejected.append(rejected_item)



        status = self.status.value

        triggers = self.triggers

        webhooks = []
        for webhooks_item_data in self.webhooks:
            webhooks_item = webhooks_item_data.to_dict()
            webhooks.append(webhooks_item)



        workflows = []
        for workflows_item_data in self.workflows:
            workflows_item = workflows_item_data.to_dict()
            workflows.append(workflows_item)



        address = self.address

        connected_at: str | Unset = UNSET
        if not isinstance(self.connected_at, Unset):
            connected_at = self.connected_at.isoformat()

        connected_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.connected_by, Unset):
            connected_by = self.connected_by.to_dict()

        connection_id = self.connection_id

        first_call: dict[str, Any] | Unset = UNSET
        if not isinstance(self.first_call, Unset):
            first_call = self.first_call.to_dict()

        key: dict[str, Any] | Unset = UNSET
        if not isinstance(self.key, Unset):
            key = self.key.to_dict()

        label = self.label

        old_key: dict[str, Any] | Unset = UNSET
        if not isinstance(self.old_key, Unset):
            old_key = self.old_key.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "actions": actions,
            "canManage": can_manage,
            "helpers": helpers,
            "instanceUrl": instance_url,
            "items": items,
            "kind": kind,
            "numbers": numbers,
            "reachability": reachability,
            "reasons": reasons,
            "receives": receives,
            "rejected": rejected,
            "status": status,
            "triggers": triggers,
            "webhooks": webhooks,
            "workflows": workflows,
        })
        if address is not UNSET:
            field_dict["address"] = address
        if connected_at is not UNSET:
            field_dict["connectedAt"] = connected_at
        if connected_by is not UNSET:
            field_dict["connectedBy"] = connected_by
        if connection_id is not UNSET:
            field_dict["connectionId"] = connection_id
        if first_call is not UNSET:
            field_dict["firstCall"] = first_call
        if key is not UNSET:
            field_dict["key"] = key
        if label is not UNSET:
            field_dict["label"] = label
        if old_key is not UNSET:
            field_dict["oldKey"] = old_key

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_first_call import IntegrationFirstCall # noqa: PLC0415
        from ..models.integration_item_state import IntegrationItemState # noqa: PLC0415
        from ..models.integration_key_view import IntegrationKeyView # noqa: PLC0415
        from ..models.integration_managed_hook import IntegrationManagedHook # noqa: PLC0415
        from ..models.integration_numbers import IntegrationNumbers # noqa: PLC0415
        from ..models.integration_person import IntegrationPerson # noqa: PLC0415
        from ..models.integration_reachability import IntegrationReachability # noqa: PLC0415
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        from ..models.integration_rejected_call import IntegrationRejectedCall # noqa: PLC0415
        from ..models.integration_workflow_row import IntegrationWorkflowRow # noqa: PLC0415
        d = dict(src_dict)
        actions = d.pop("actions")

        can_manage = d.pop("canManage")

        helpers = cast(list[str], d.pop("helpers"))


        instance_url = d.pop("instanceUrl")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = IntegrationItemState.from_dict(items_item_data)



            items.append(items_item)


        kind = IntegrationAppKind(d.pop("kind"))




        numbers = IntegrationNumbers.from_dict(d.pop("numbers"))




        reachability = IntegrationReachability.from_dict(d.pop("reachability"))




        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = IntegrationReason.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        receives = cast(list[str], d.pop("receives"))


        rejected = []
        _rejected = d.pop("rejected")
        for rejected_item_data in (_rejected):
            rejected_item = IntegrationRejectedCall.from_dict(rejected_item_data)



            rejected.append(rejected_item)


        status = IntegrationAppStatus(d.pop("status"))




        triggers = d.pop("triggers")

        webhooks = []
        _webhooks = d.pop("webhooks")
        for webhooks_item_data in (_webhooks):
            webhooks_item = IntegrationManagedHook.from_dict(webhooks_item_data)



            webhooks.append(webhooks_item)


        workflows = []
        _workflows = d.pop("workflows")
        for workflows_item_data in (_workflows):
            workflows_item = IntegrationWorkflowRow.from_dict(workflows_item_data)



            workflows.append(workflows_item)


        address = d.pop("address", UNSET)

        _connected_at = d.pop("connectedAt", UNSET)
        connected_at: datetime.datetime | Unset
        if isinstance(_connected_at,  Unset):
            connected_at = UNSET
        else:
            connected_at = datetime.datetime.fromisoformat(_connected_at)




        _connected_by = d.pop("connectedBy", UNSET)
        connected_by: IntegrationPerson | Unset
        if isinstance(_connected_by,  Unset):
            connected_by = UNSET
        else:
            connected_by = IntegrationPerson.from_dict(_connected_by)




        connection_id = d.pop("connectionId", UNSET)

        _first_call = d.pop("firstCall", UNSET)
        first_call: IntegrationFirstCall | Unset
        if isinstance(_first_call,  Unset):
            first_call = UNSET
        else:
            first_call = IntegrationFirstCall.from_dict(_first_call)




        _key = d.pop("key", UNSET)
        key: IntegrationKeyView | Unset
        if isinstance(_key,  Unset):
            key = UNSET
        else:
            key = IntegrationKeyView.from_dict(_key)




        label = d.pop("label", UNSET)

        _old_key = d.pop("oldKey", UNSET)
        old_key: IntegrationKeyView | Unset
        if isinstance(_old_key,  Unset):
            old_key = UNSET
        else:
            old_key = IntegrationKeyView.from_dict(_old_key)




        integration_app = cls(
            actions=actions,
            can_manage=can_manage,
            helpers=helpers,
            instance_url=instance_url,
            items=items,
            kind=kind,
            numbers=numbers,
            reachability=reachability,
            reasons=reasons,
            receives=receives,
            rejected=rejected,
            status=status,
            triggers=triggers,
            webhooks=webhooks,
            workflows=workflows,
            address=address,
            connected_at=connected_at,
            connected_by=connected_by,
            connection_id=connection_id,
            first_call=first_call,
            key=key,
            label=label,
            old_key=old_key,
        )

        return integration_app

