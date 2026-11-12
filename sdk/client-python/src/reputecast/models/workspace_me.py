from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.workspace_me_actor_type import WorkspaceMeActorType
from ..models.workspace_me_hidden_sections_item import WorkspaceMeHiddenSectionsItem
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.me_workspace import MeWorkspace
  from ..models.scope import Scope
  from ..models.shell_host_repute import ShellHostRepute





T = TypeVar("T", bound="WorkspaceMe")



@_attrs_define
class WorkspaceMe:
    """ 
        Attributes:
            actor_type (WorkspaceMeActorType):
            hidden_sections (list[WorkspaceMeHiddenSectionsItem]):
            permissions (list[str]):
            scope (Scope):
            timezone (str):
            unread (int):
            workspace (MeWorkspace):
            host_repute (ShellHostRepute | Unset):
     """

    actor_type: WorkspaceMeActorType
    hidden_sections: list[WorkspaceMeHiddenSectionsItem]
    permissions: list[str]
    scope: Scope
    timezone: str
    unread: int
    workspace: MeWorkspace
    host_repute: ShellHostRepute | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.me_workspace import MeWorkspace # noqa: PLC0415
        from ..models.scope import Scope # noqa: PLC0415
        from ..models.shell_host_repute import ShellHostRepute # noqa: PLC0415
        actor_type = self.actor_type.value

        hidden_sections = []
        for hidden_sections_item_data in self.hidden_sections:
            hidden_sections_item = hidden_sections_item_data.value
            hidden_sections.append(hidden_sections_item)



        permissions = self.permissions



        scope = self.scope.to_dict()

        timezone = self.timezone

        unread = self.unread

        workspace = self.workspace.to_dict()

        host_repute: dict[str, Any] | Unset = UNSET
        if not isinstance(self.host_repute, Unset):
            host_repute = self.host_repute.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "actorType": actor_type,
            "hiddenSections": hidden_sections,
            "permissions": permissions,
            "scope": scope,
            "timezone": timezone,
            "unread": unread,
            "workspace": workspace,
        })
        if host_repute is not UNSET:
            field_dict["hostRepute"] = host_repute

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.me_workspace import MeWorkspace # noqa: PLC0415
        from ..models.scope import Scope # noqa: PLC0415
        from ..models.shell_host_repute import ShellHostRepute # noqa: PLC0415
        d = dict(src_dict)
        actor_type = WorkspaceMeActorType(d.pop("actorType"))




        hidden_sections = []
        _hidden_sections = d.pop("hiddenSections")
        for hidden_sections_item_data in (_hidden_sections):
            hidden_sections_item = WorkspaceMeHiddenSectionsItem(hidden_sections_item_data)



            hidden_sections.append(hidden_sections_item)


        permissions = cast(list[str], d.pop("permissions"))


        scope = Scope.from_dict(d.pop("scope"))




        timezone = d.pop("timezone")

        unread = d.pop("unread")

        workspace = MeWorkspace.from_dict(d.pop("workspace"))




        _host_repute = d.pop("hostRepute", UNSET)
        host_repute: ShellHostRepute | Unset
        if isinstance(_host_repute,  Unset):
            host_repute = UNSET
        else:
            host_repute = ShellHostRepute.from_dict(_host_repute)




        workspace_me = cls(
            actor_type=actor_type,
            hidden_sections=hidden_sections,
            permissions=permissions,
            scope=scope,
            timezone=timezone,
            unread=unread,
            workspace=workspace,
            host_repute=host_repute,
        )

        return workspace_me

