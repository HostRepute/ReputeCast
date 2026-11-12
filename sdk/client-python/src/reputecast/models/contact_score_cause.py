from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_score_cause_kind import ContactScoreCauseKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="ContactScoreCause")



@_attrs_define
class ContactScoreCause:
    """ 
        Attributes:
            kind (ContactScoreCauseKind):
            email_name (str | Unset):
            event_name (str | Unset):
            form_name (str | Unset):
            link_label (str | Unset):
            list_name (str | Unset):
            url (str | Unset):
     """

    kind: ContactScoreCauseKind
    email_name: str | Unset = UNSET
    event_name: str | Unset = UNSET
    form_name: str | Unset = UNSET
    link_label: str | Unset = UNSET
    list_name: str | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        email_name = self.email_name

        event_name = self.event_name

        form_name = self.form_name

        link_label = self.link_label

        list_name = self.list_name

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if email_name is not UNSET:
            field_dict["emailName"] = email_name
        if event_name is not UNSET:
            field_dict["eventName"] = event_name
        if form_name is not UNSET:
            field_dict["formName"] = form_name
        if link_label is not UNSET:
            field_dict["linkLabel"] = link_label
        if list_name is not UNSET:
            field_dict["listName"] = list_name
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = ContactScoreCauseKind(d.pop("kind"))




        email_name = d.pop("emailName", UNSET)

        event_name = d.pop("eventName", UNSET)

        form_name = d.pop("formName", UNSET)

        link_label = d.pop("linkLabel", UNSET)

        list_name = d.pop("listName", UNSET)

        url = d.pop("url", UNSET)

        contact_score_cause = cls(
            kind=kind,
            email_name=email_name,
            event_name=event_name,
            form_name=form_name,
            link_label=link_label,
            list_name=list_name,
            url=url,
        )

        return contact_score_cause

