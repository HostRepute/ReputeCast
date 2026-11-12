from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.workflow_use_opt_in import WorkflowUseOptIn
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="WorkflowUse")



@_attrs_define
class WorkflowUse:
    """ 
        Attributes:
            item (str):
            event (str | Unset):
            goal_of (list[str] | Unset):
            lists (list[str] | Unset):
            opt_in (WorkflowUseOptIn | Unset):
            starts (list[str] | Unset):
            tags (list[str] | Unset):
            template (str | Unset):
     """

    item: str
    event: str | Unset = UNSET
    goal_of: list[str] | Unset = UNSET
    lists: list[str] | Unset = UNSET
    opt_in: WorkflowUseOptIn | Unset = UNSET
    starts: list[str] | Unset = UNSET
    tags: list[str] | Unset = UNSET
    template: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        item = self.item

        event = self.event

        goal_of: list[str] | Unset = UNSET
        if not isinstance(self.goal_of, Unset):
            goal_of = self.goal_of



        lists: list[str] | Unset = UNSET
        if not isinstance(self.lists, Unset):
            lists = self.lists



        opt_in: str | Unset = UNSET
        if not isinstance(self.opt_in, Unset):
            opt_in = self.opt_in.value


        starts: list[str] | Unset = UNSET
        if not isinstance(self.starts, Unset):
            starts = self.starts



        tags: list[str] | Unset = UNSET
        if not isinstance(self.tags, Unset):
            tags = self.tags



        template = self.template


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "item": item,
        })
        if event is not UNSET:
            field_dict["event"] = event
        if goal_of is not UNSET:
            field_dict["goalOf"] = goal_of
        if lists is not UNSET:
            field_dict["lists"] = lists
        if opt_in is not UNSET:
            field_dict["optIn"] = opt_in
        if starts is not UNSET:
            field_dict["starts"] = starts
        if tags is not UNSET:
            field_dict["tags"] = tags
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        item = d.pop("item")

        event = d.pop("event", UNSET)

        goal_of = cast(list[str], d.pop("goalOf", UNSET))


        lists = cast(list[str], d.pop("lists", UNSET))


        _opt_in = d.pop("optIn", UNSET)
        opt_in: WorkflowUseOptIn | Unset
        if isinstance(_opt_in,  Unset):
            opt_in = UNSET
        else:
            opt_in = WorkflowUseOptIn(_opt_in)




        starts = cast(list[str], d.pop("starts", UNSET))


        tags = cast(list[str], d.pop("tags", UNSET))


        template = d.pop("template", UNSET)

        workflow_use = cls(
            item=item,
            event=event,
            goal_of=goal_of,
            lists=lists,
            opt_in=opt_in,
            starts=starts,
            tags=tags,
            template=template,
        )

        return workflow_use

