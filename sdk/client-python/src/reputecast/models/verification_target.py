from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_target_kind import VerificationTargetKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_filter import ContactFilter





T = TypeVar("T", bound="VerificationTarget")



@_attrs_define
class VerificationTarget:
    """ 
        Attributes:
            kind (VerificationTargetKind):
            contact_id (str | Unset):
            contact_ids (list[str] | Unset):
            filter_ (ContactFilter | Unset):
            import_id (str | Unset):
            label (str | Unset):
            list_id (str | Unset):
            segment_id (str | Unset):
            tiers (list[str] | Unset):
     """

    kind: VerificationTargetKind
    contact_id: str | Unset = UNSET
    contact_ids: list[str] | Unset = UNSET
    filter_: ContactFilter | Unset = UNSET
    import_id: str | Unset = UNSET
    label: str | Unset = UNSET
    list_id: str | Unset = UNSET
    segment_id: str | Unset = UNSET
    tiers: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        kind = self.kind.value

        contact_id = self.contact_id

        contact_ids: list[str] | Unset = UNSET
        if not isinstance(self.contact_ids, Unset):
            contact_ids = self.contact_ids



        filter_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.filter_, Unset):
            filter_ = self.filter_.to_dict()

        import_id = self.import_id

        label = self.label

        list_id = self.list_id

        segment_id = self.segment_id

        tiers: list[str] | Unset = UNSET
        if not isinstance(self.tiers, Unset):
            tiers = self.tiers




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if contact_ids is not UNSET:
            field_dict["contactIds"] = contact_ids
        if filter_ is not UNSET:
            field_dict["filter"] = filter_
        if import_id is not UNSET:
            field_dict["importId"] = import_id
        if label is not UNSET:
            field_dict["label"] = label
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if segment_id is not UNSET:
            field_dict["segmentId"] = segment_id
        if tiers is not UNSET:
            field_dict["tiers"] = tiers

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        d = dict(src_dict)
        kind = VerificationTargetKind(d.pop("kind"))




        contact_id = d.pop("contactId", UNSET)

        contact_ids = cast(list[str], d.pop("contactIds", UNSET))


        _filter_ = d.pop("filter", UNSET)
        filter_: ContactFilter | Unset
        if isinstance(_filter_,  Unset):
            filter_ = UNSET
        else:
            filter_ = ContactFilter.from_dict(_filter_)




        import_id = d.pop("importId", UNSET)

        label = d.pop("label", UNSET)

        list_id = d.pop("listId", UNSET)

        segment_id = d.pop("segmentId", UNSET)

        tiers = cast(list[str], d.pop("tiers", UNSET))


        verification_target = cls(
            kind=kind,
            contact_id=contact_id,
            contact_ids=contact_ids,
            filter_=filter_,
            import_id=import_id,
            label=label,
            list_id=list_id,
            segment_id=segment_id,
            tiers=tiers,
        )

        return verification_target

