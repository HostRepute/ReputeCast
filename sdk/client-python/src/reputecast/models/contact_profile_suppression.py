from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_profile_suppression_scope import ContactProfileSuppressionScope
from typing import cast
import datetime






T = TypeVar("T", bound="ContactProfileSuppression")



@_attrs_define
class ContactProfileSuppression:
    """ 
        Attributes:
            at (datetime.datetime):
            id (str):
            reason (str):
            scope (ContactProfileSuppressionScope):
     """

    at: datetime.datetime
    id: str
    reason: str
    scope: ContactProfileSuppressionScope





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        id = self.id

        reason = self.reason

        scope = self.scope.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "id": id,
            "reason": reason,
            "scope": scope,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        id = d.pop("id")

        reason = d.pop("reason")

        scope = ContactProfileSuppressionScope(d.pop("scope"))




        contact_profile_suppression = cls(
            at=at,
            id=id,
            reason=reason,
            scope=scope,
        )

        return contact_profile_suppression

