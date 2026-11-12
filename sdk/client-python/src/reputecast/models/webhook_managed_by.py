from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WebhookManagedBy")



@_attrs_define
class WebhookManagedBy:
    """ 
        Attributes:
            integration_id (str):
            kind (str):  Example: n8n.
            name (str):  Example: n8n.
            endpoint (str | Unset):  Example: https://n8n.acme.io/webhook/7c1e…/reputecast.
            workflow (str | Unset):
     """

    integration_id: str
    kind: str
    name: str
    endpoint: str | Unset = UNSET
    workflow: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        integration_id = self.integration_id

        kind = self.kind

        name = self.name

        endpoint = self.endpoint

        workflow = self.workflow


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "integrationId": integration_id,
            "kind": kind,
            "name": name,
        })
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint
        if workflow is not UNSET:
            field_dict["workflow"] = workflow

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        integration_id = d.pop("integrationId")

        kind = d.pop("kind")

        name = d.pop("name")

        endpoint = d.pop("endpoint", UNSET)

        workflow = d.pop("workflow", UNSET)

        webhook_managed_by = cls(
            integration_id=integration_id,
            kind=kind,
            name=name,
            endpoint=endpoint,
            workflow=workflow,
        )

        return webhook_managed_by

