from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_me_integration import IntegrationMeIntegration
from typing import cast






T = TypeVar("T", bound="IntegrationMe")



@_attrs_define
class IntegrationMe:
    """ 
        Attributes:
            actions (list[str]): Actions turned on
            integration (IntegrationMeIntegration):
            key (str): The key's name
            permissions (list[str]):
            triggers (list[str]): Events of the triggers turned on
            workspace (str): The workspace slug, for the other calls' paths
            workspace_name (str):
     """

    actions: list[str]
    integration: IntegrationMeIntegration
    key: str
    permissions: list[str]
    triggers: list[str]
    workspace: str
    workspace_name: str





    def to_dict(self) -> dict[str, Any]:
        actions = self.actions



        integration = self.integration.value

        key = self.key

        permissions = self.permissions



        triggers = self.triggers



        workspace = self.workspace

        workspace_name = self.workspace_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "actions": actions,
            "integration": integration,
            "key": key,
            "permissions": permissions,
            "triggers": triggers,
            "workspace": workspace,
            "workspaceName": workspace_name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        actions = cast(list[str], d.pop("actions"))


        integration = IntegrationMeIntegration(d.pop("integration"))




        key = d.pop("key")

        permissions = cast(list[str], d.pop("permissions"))


        triggers = cast(list[str], d.pop("triggers"))


        workspace = d.pop("workspace")

        workspace_name = d.pop("workspaceName")

        integration_me = cls(
            actions=actions,
            integration=integration,
            key=key,
            permissions=permissions,
            triggers=triggers,
            workspace=workspace,
            workspace_name=workspace_name,
        )

        return integration_me

