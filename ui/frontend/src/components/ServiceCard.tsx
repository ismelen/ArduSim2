import { type DeployedService } from '../hooks/useFleet';

interface ServiceCardProps {
  service: DeployedService;
  isEditing: boolean;
  onEdit: () => void;
  onDelete: () => void;
  renderEditForm: () => React.ReactNode;
  onCancelEdit: () => void;
  onUpdate: () => void;
}

export const ServiceCard: React.FC<ServiceCardProps> = ({
  service,
  isEditing,
  onEdit,
  onDelete,
  renderEditForm,
  onCancelEdit,
  onUpdate,
}) => {
  return (
    <div className={`service-card ${isEditing ? 'editing' : ''}`}>
      {isEditing ? (
        <div className="service-card-edit">
          <div className="service-card-edit-title display-font">Edit: {service.serviceTitle}</div>
          {renderEditForm()}
          <div className="service-card-edit-footer">
            <button className="outline-btn danger-outline" onClick={onCancelEdit}>Cancel</button>
            <button className="deploy-btn" onClick={onUpdate} style={{ padding: '0.5rem 1.25rem' }}>Update</button>
          </div>
        </div>
      ) : (
        <div className="service-card-row">
          <h3 className="service-card-title display-font">{service.serviceTitle}</h3>
          <div className="service-card-actions">
            <button className="material-symbols-outlined" onClick={onEdit}>edit</button>
            <button className="material-symbols-outlined" onClick={onDelete}>delete</button>
          </div>
        </div>
      )}
    </div>
  );
};
