from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="TransactionalTemplateAccess")



@_attrs_define
class TransactionalTemplateAccess:
    """ 
        Attributes:
            can_edit (bool):
            can_publish (bool):
            can_send_test_to_others (bool):
            can_view_content (bool):
            publish_helpers (list[str]):
            edit_reason (str | Unset):
            publish_reason (str | Unset):
     """

    can_edit: bool
    can_publish: bool
    can_send_test_to_others: bool
    can_view_content: bool
    publish_helpers: list[str]
    edit_reason: str | Unset = UNSET
    publish_reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        can_edit = self.can_edit

        can_publish = self.can_publish

        can_send_test_to_others = self.can_send_test_to_others

        can_view_content = self.can_view_content

        publish_helpers = self.publish_helpers



        edit_reason = self.edit_reason

        publish_reason = self.publish_reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canPublish": can_publish,
            "canSendTestToOthers": can_send_test_to_others,
            "canViewContent": can_view_content,
            "publishHelpers": publish_helpers,
        })
        if edit_reason is not UNSET:
            field_dict["editReason"] = edit_reason
        if publish_reason is not UNSET:
            field_dict["publishReason"] = publish_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_publish = d.pop("canPublish")

        can_send_test_to_others = d.pop("canSendTestToOthers")

        can_view_content = d.pop("canViewContent")

        publish_helpers = cast(list[str], d.pop("publishHelpers"))


        edit_reason = d.pop("editReason", UNSET)

        publish_reason = d.pop("publishReason", UNSET)

        transactional_template_access = cls(
            can_edit=can_edit,
            can_publish=can_publish,
            can_send_test_to_others=can_send_test_to_others,
            can_view_content=can_view_content,
            publish_helpers=publish_helpers,
            edit_reason=edit_reason,
            publish_reason=publish_reason,
        )

        return transactional_template_access

